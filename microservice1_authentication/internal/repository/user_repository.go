package repository

import (
	"errors"
	"fmt"

	"github.com/appointy/notli/microservice1_authentication/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository handles database operations for users
type UserRepository struct {
	DB *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

// CreateUser adds a new user to the database
func (r *UserRepository) CreateUser(user *models.User) error {
	// Check if email already exists
	var existingUser models.User
	if err := r.DB.Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
		// Email already exists
		return fmt.Errorf("email '%s' already registered", user.Email)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Other database error during check
		return fmt.Errorf("failed to check for existing email: %w", err)
	}

	// Create the user
	if err := r.DB.Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUserByEmail retrieves a user by their email address
func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user with email '%s' not found", email)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

// GetUserByID retrieves a user by their ID
func (r *UserRepository) GetUserByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.DB.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user with id '%s' not found", id)
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &user, nil
}
