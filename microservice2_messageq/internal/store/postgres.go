package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyStore defines the interface for checking idempotency.
type IdempotencyStore interface {
	Check(ctx context.Context, key string) (isDuplicate bool, err error)
	Disconnect(ctx context.Context) error
}

// postgresStore implements the IdempotencyStore interface using PostgreSQL with in-memory caching.
type postgresStore struct {
	pool          *pgxpool.Pool
	cache         map[string]bool
	cacheMu       sync.RWMutex  // Mutex for cache access
	lockMap       sync.Map      // For distributed locking to prevent race conditions
	useLocalCache bool
}

const (
	// PostgreSQL unique violation error code
	pgErrUniqueViolation = "23505"
)

// NewPostgresStore creates a new PostgreSQL-backed IdempotencyStore.
// It initializes the connection pool and ensures the idempotency table exists.
func NewPostgresStore(ctx context.Context) (IdempotencyStore, error) {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD") // Password might be empty
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "appointy"
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres connection string: %w", err)
	}

	// Set optimized connection pool settings for high concurrency
	config.MaxConns = 100                // Increased from 10 to handle more concurrent connections
	config.MinConns = 20                 // Increased to keep more connections ready
	config.MaxConnIdleTime = 30 * time.Second // Decreased to recycle connections faster
	config.MaxConnLifetime = 30 * time.Minute // Decreased to prevent connection staleness
	config.HealthCheckPeriod = 30 * time.Second // More frequent health checks
	
	// Set connection timeout through ConnConfig
	config.ConnConfig.ConnectTimeout = 5 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL!")

	if err := ensureTableExists(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ensure idempotency table exists: %w", err)
	}

	// Determine if we should use local cache (default to true)
	useCache := true
	if os.Getenv("DISABLE_IDEMPOTENCY_CACHE") == "true" {
		useCache = false
		log.Println("Warning: Idempotency cache is disabled")
	}
	
	// Create store with our custom map-based cache
	store := &postgresStore{
		pool:          pool,
		cache:         make(map[string]bool),
		useLocalCache: useCache,
	}
	
	log.Println("PostgreSQL store initialized with high-performance connection pool and in-memory caching")
	return store, nil
}

// ensureTableExists creates the idempotency_keys table if it doesn't exist.
func ensureTableExists(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS idempotency_keys (
		id VARCHAR(128) PRIMARY KEY,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	`
	_, err := pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to execute create table statement: %w", err)
	}
	log.Println("Table 'idempotency_keys' checked/created successfully.")
	return nil
}

// Check checks if the key exists with optimized performance using caching.
// Returns true if exists (duplicate), false otherwise.
func (s *postgresStore) Check(ctx context.Context, key string) (bool, error) {
	// Fast path: Check in-memory cache first for significant performance improvement
	if s.useLocalCache {
		s.cacheMu.RLock()
		_, found := s.cache[key]
		s.cacheMu.RUnlock()
		
		if found {
			// Key exists in cache, it's a duplicate
			return true, nil
		}
	}
	
	// Use a mutex-per-key approach to prevent race conditions on same key
	// Get or create lock for this key
	lockInterface, _ := s.lockMap.LoadOrStore(key, &sync.Mutex{})
	lock := lockInterface.(*sync.Mutex)
	
	// Lock to prevent concurrent DB operations for the same key
	lock.Lock()
	defer func() {
		lock.Unlock()
		// Clean up the lock if we're done with it
		s.lockMap.Delete(key)
	}()
	
	// Double-check cache after acquiring lock (another thread may have updated cache)
	if s.useLocalCache {
		s.cacheMu.RLock()
		_, found := s.cache[key]
		s.cacheMu.RUnlock()
		
		if found {
			return true, nil
		}
	}
	
	// Slow path: Check database using an optimized query
	query := `INSERT INTO idempotency_keys (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`
	
	// Use a shorter timeout for the database operation
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	
	commandTag, err := s.pool.Exec(dbCtx, query, key)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			log.Printf("PostgreSQL error during idempotency check (Code: %s): %v", pgErr.Code, err)
		} else {
			log.Printf("Error checking idempotency for key '%s': %v", key, err)
		}
		return false, fmt.Errorf("database error during idempotency check")
	}
	
	isDuplicate := commandTag.RowsAffected() == 0
	
	// Cache the result for future lookups
	if s.useLocalCache {
		s.cacheMu.Lock()
		s.cache[key] = true
		s.cacheMu.Unlock()
	}
	
	return isDuplicate, nil
}

// Disconnect closes the PostgreSQL connection pool.
func (s *postgresStore) Disconnect(ctx context.Context) error {
	if s.pool != nil {
		log.Println("Disconnecting from PostgreSQL...")
		s.pool.Close()
	}
	return nil
}
