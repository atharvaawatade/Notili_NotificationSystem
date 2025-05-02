package repository

import (
	"fmt"

	"github.com/appointy/notli/microservice1_authentication/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIKeyRepository handles database operations for API keys
type APIKeyRepository struct {
	DB *gorm.DB
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository {
	return &APIKeyRepository{DB: db}
}

// CreateAPIKey adds a new API key to the database
func (r *APIKeyRepository) CreateAPIKey(apiKey *models.APIKey) error {
	if err := r.DB.Create(apiKey).Error; err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}
	return nil
}

// GetAPIKeysByUserID retrieves all API keys for a specific user
func (r *APIKeyRepository) GetAPIKeysByUserID(userID uuid.UUID) ([]models.APIKey, error) {
	var apiKeys []models.APIKey
	if err := r.DB.Where("user_id = ?", userID).Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get API keys for user %s: %w", userID, err)
	}
	return apiKeys, nil
}

// GetAPIKeyByID retrieves a specific API key by its ID and User ID
func (r *APIKeyRepository) GetAPIKeyByID(id uuid.UUID, userID uuid.UUID) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.DB.Where("id = ? AND user_id = ?", id, userID).First(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("failed to get API key %s for user %s: %w", id, userID, err)
	}
	return &apiKey, nil
}

// DeleteAPIKey removes an API key from the database
func (r *APIKeyRepository) DeleteAPIKey(id uuid.UUID, userID uuid.UUID) error {
	// Ensure the key belongs to the user before deleting
	result := r.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.APIKey{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete API key %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("API key %s not found or does not belong to user %s", id, userID)
	}
	return nil
}

// FindAPIKeyByHash retrieves an API key by its hash (used for key validation)
// Note: This doesn't check user ID, as it's for validating keys from other services.
func (r *APIKeyRepository) FindAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.DB.Preload("User").Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("failed to find API key by hash: %w", err)
	}
	// Optionally: Update LastUsed time here if needed
	return &apiKey, nil
}

// GetAPIKeysByPrefix retrieves all API keys with a specific prefix
func (r *APIKeyRepository) GetAPIKeysByPrefix(prefix string) ([]models.APIKey, error) {
	var apiKeys []models.APIKey
	if err := r.DB.Where("prefix = ?", prefix).Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get API keys by prefix %s: %w", prefix, err)
	}
	return apiKeys, nil
}

// GetUserByAPIKeyID retrieves the user associated with an API key
func (r *APIKeyRepository) GetUserByAPIKeyID(keyID uuid.UUID) (*models.User, error) {
	var apiKey models.APIKey
	if err := r.DB.Where("id = ?", keyID).First(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("failed to find API key with ID %s: %w", keyID, err)
	}
	
	var user models.User
	if err := r.DB.Where("id = ?", apiKey.UserID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to find user for API key %s: %w", keyID, err)
	}
	
	return &user, nil
}

// UpdateAPIKey updates an existing API key
func (r *APIKeyRepository) UpdateAPIKey(apiKey *models.APIKey) error {
	if err := r.DB.Save(apiKey).Error; err != nil {
		return fmt.Errorf("failed to update API key %s: %w", apiKey.ID, err)
	}
	return nil
}
