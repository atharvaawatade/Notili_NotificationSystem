package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIKey represents an API key for a user
type APIKey struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"` // Foreign key to User
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`    // User-defined name for the key
	Prefix    string    `gorm:"type:varchar(10);not null;index" json:"prefix"` // e.g., "sk_live_"
	KeyHash   string    `gorm:"type:varchar(255);uniqueIndex:idx_apikeys_keyhash;not null" json:"-"` // Explicitly named index
	LastUsed  *time.Time `json:"last_used,omitempty"`                    // Pointer to allow null
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Association (optional but good practice for GORM)
	User User `gorm:"foreignKey:UserID"`
}

// BeforeCreate ensures a UUID is set if not provided
func (apiKey *APIKey) BeforeCreate(tx *gorm.DB) (err error) {
	if apiKey.ID == uuid.Nil {
		apiKey.ID = uuid.New()
	}
	return
}
