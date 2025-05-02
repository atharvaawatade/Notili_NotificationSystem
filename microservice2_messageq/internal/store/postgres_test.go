package store

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testStore IdempotencyStore

// TestMain sets up the test database connection and ensures cleanup.
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var err error
	testStore, err = NewPostgresStore(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v. Ensure DB is running and env vars (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME) are set.", err)
	}

	// Run tests
	exitCode := m.Run()

	// Teardown: Disconnect from DB
	// Use a new context for cleanup
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cleanupCancel()
	if err := testStore.Disconnect(cleanupCtx); err != nil {
		log.Printf("Warning: failed to disconnect test database: %v", err)
	}

	os.Exit(exitCode)
}

// Helper to clean up test data
func cleanupKey(t *testing.T, key string) {
	pgStore, ok := testStore.(*postgresStore)
	if !ok || pgStore.pool == nil {
		t.Fatal("Test store is not a valid postgresStore or pool is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := pgStore.pool.Exec(ctx, "DELETE FROM idempotency_keys WHERE id = $1", key)
	if err != nil {
		t.Logf("Warning: failed to clean up idempotency key '%s': %v", key, err)
	}
}

func TestPostgresCheckIdempotency(t *testing.T) {
	require.NotNil(t, testStore, "Test store should be initialized")
	ctx := context.Background()

	// --- Test Case 1: New Key --- //
	newKey := fmt.Sprintf("test-idem-key-pg-%d", time.Now().UnixNano())
	t.Logf("Testing with new key: %s", newKey)
	// Ensure cleanup happens even if test fails
	t.Cleanup(func() { cleanupKey(t, newKey) })

	isDuplicate, err := testStore.Check(ctx, newKey)
	assert.NoError(t, err, "Check failed for new key")
	assert.False(t, isDuplicate, "Check should return false for a new key")

	// Verify the key was actually inserted (optional check)
	pgStore, ok := testStore.(*postgresStore)
	require.True(t, ok, "testStore should be a *postgresStore")
	var count int
	findCtx, findCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer findCancel()
	err = pgStore.pool.QueryRow(findCtx, "SELECT COUNT(*) FROM idempotency_keys WHERE id = $1", newKey).Scan(&count)
	assert.NoError(t, err, "Failed to query the newly inserted key")
	assert.Equal(t, 1, count, "Expected to find 1 record for the new key")

	// --- Test Case 2: Duplicate Key --- //
	t.Logf("Testing with duplicate key: %s", newKey)
	isDuplicateAgain, errAgain := testStore.Check(ctx, newKey)
	assert.NoError(t, errAgain, "Check failed for duplicate key check")
	assert.True(t, isDuplicateAgain, "Check should return true for a duplicate key")

	// Verify count is still 1
	count = 0 // Reset count
	findCtx2, findCancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer findCancel2()
	err = pgStore.pool.QueryRow(findCtx2, "SELECT COUNT(*) FROM idempotency_keys WHERE id = $1", newKey).Scan(&count)
	assert.NoError(t, err, "Failed to query the duplicate key")
	assert.Equal(t, 1, count, "Expected count to remain 1 after duplicate check")

	// --- Test Case 3: Empty Key (should fail at DB level or be caught by validation) --- //
	// PostgreSQL might reject empty string for PRIMARY KEY depending on exact type/constraints
	// Our validation in handler should prevent this anyway.
	// _, errEmpty := testStore.Check(ctx, "")
	// assert.Error(t, errEmpty, "Check should ideally return an error for an empty key, or DB error")
}
