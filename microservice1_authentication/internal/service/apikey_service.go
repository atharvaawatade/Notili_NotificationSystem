package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/appointy/notli/microservice1_authentication/internal/models"
	"github.com/appointy/notli/microservice1_authentication/internal/repository"
	"github.com/appointy/notli/microservice1_authentication/internal/utils"
	"github.com/google/uuid"
)

const (
	apiKeyLength = 32 // Length of the random part of the API key in bytes
	apiKeyPrefix = "ntli_" // Prefix for all NOTLI keys
)

// APIKeyService handles business logic for API keys
type APIKeyService struct {
	APIKeyRepo *repository.APIKeyRepository
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(apiKeyRepo *repository.APIKeyRepository) *APIKeyService {
	return &APIKeyService{APIKeyRepo: apiKeyRepo}
}

// generateSecureKey generates a secure random API key string and its prefix
func generateSecureKey() (string, string, error) {
	b := make([]byte, apiKeyLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes for API key: %w", err)
	}
	// Use RawURLEncoding to avoid '+' and '/' characters, making it URL-safe
	key := base64.RawURLEncoding.EncodeToString(b)
	fullKey := apiKeyPrefix + key
	return fullKey, apiKeyPrefix, nil
}

// CreateAPIKey generates a new API key for a user
func (s *APIKeyService) CreateAPIKey(userID uuid.UUID, name string) (string, *models.APIKey, error) {
	if name == "" {
		return "", nil, fmt.Errorf("API key name cannot be empty")
	}

	fullKey, prefix, err := generateSecureKey()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate secure key: %w", err)
	}

	// Hash the *full* key for storage
	keyHash, err := utils.HashPassword(fullKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to hash API key: %w", err)
	}

	apiKey := &models.APIKey{
		UserID:  userID,
		Name:    name,
		Prefix:  prefix,
		KeyHash: keyHash,
		// LastUsed starts as nil
	}

	if err := s.APIKeyRepo.CreateAPIKey(apiKey); err != nil {
		// Check if the error is due to a duplicate key hash (extremely unlikely, but good practice)
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			// Potentially retry generation or return a specific error
			return "", nil, fmt.Errorf("API key hash collision, please try again")
		}
		return "", nil, err // Other creation error
	}

	// Return the *full, unhashed* key only once upon creation
	return fullKey, apiKey, nil
}

// GetAPIKeys retrieves all API keys for a user
func (s *APIKeyService) GetAPIKeys(userID uuid.UUID) ([]models.APIKey, error) {
	return s.APIKeyRepo.GetAPIKeysByUserID(userID)
}

// DeleteAPIKey removes an API key for a user
func (s *APIKeyService) DeleteAPIKey(id uuid.UUID, userID uuid.UUID) error {
	return s.APIKeyRepo.DeleteAPIKey(id, userID)
}

// ValidateAPIKey checks if an API key is valid and returns the associated user
func (s *APIKeyService) ValidateAPIKey(keyString string) (bool, *models.User, error) {
    if keyString == "" {
        return false, nil, fmt.Errorf("API key cannot be empty")
    }

    // Extract the prefix (first few characters of the key)
    if len(keyString) < len(apiKeyPrefix) {
        return false, nil, fmt.Errorf("invalid API key format")
    }
    
    prefix := keyString[:len(apiKeyPrefix)]
    if prefix != apiKeyPrefix {
        return false, nil, fmt.Errorf("invalid API key prefix")
    }

    // Find keys with matching prefix
    keys, err := s.APIKeyRepo.GetAPIKeysByPrefix(prefix)
    if err != nil {
        return false, nil, fmt.Errorf("database error: %w", err)
    }

    // Compare the key hash
    for _, key := range keys {
        if utils.CheckPasswordHash(keyString, key.KeyHash) {
            // Get the associated user
            user, err := s.APIKeyRepo.GetUserByAPIKeyID(key.ID)
            if err != nil {
                return true, nil, fmt.Errorf("key valid but error fetching user: %w", err)
            }

            // Update LastUsed timestamp
            now := time.Now()
            key.LastUsed = &now
            err = s.APIKeyRepo.UpdateAPIKey(&key)
            if err != nil {
                // Log the error but don't fail the validation
                log.Printf("Failed to update LastUsed for API key %s: %v", key.ID, err)
            }

            return true, user, nil
        }
    }

    return false, nil, nil // Key not found or invalid
}

