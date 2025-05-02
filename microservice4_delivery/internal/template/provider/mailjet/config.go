package mailjet

// Config represents configuration for the Mailjet template adapter
type Config struct {
	APIKey    string `json:"api_key"`
	SecretKey string `json:"secret_key"`
	
	// Optional configurations
	TemplateOwnerType string `json:"template_owner_type,omitempty"` // Default: "user"
	DefaultLimit      int    `json:"default_limit,omitempty"`       // Default: 100
}

// Validate ensures all required fields are present
func (c *Config) Validate() error {
	if c.APIKey == "" || c.SecretKey == "" {
		return ErrMissingCredentials
	}
	
	// Set defaults if not provided
	if c.TemplateOwnerType == "" {
		c.TemplateOwnerType = "user"
	}
	
	if c.DefaultLimit <= 0 {
		c.DefaultLimit = 100
	}
	
	return nil
}

// ToMap converts the Config to a generic map
func (c *Config) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"api_key":             c.APIKey,
		"secret_key":          c.SecretKey,
		"template_owner_type": c.TemplateOwnerType,
		"default_limit":       c.DefaultLimit,
	}
}

// FromMap populates the Config from a generic map
func (c *Config) FromMap(m map[string]interface{}) {
	if apiKey, ok := m["api_key"].(string); ok {
		c.APIKey = apiKey
	}
	
	if secretKey, ok := m["secret_key"].(string); ok {
		c.SecretKey = secretKey
	}
	
	if ownerType, ok := m["template_owner_type"].(string); ok {
		c.TemplateOwnerType = ownerType
	}
	
	if limit, ok := m["default_limit"].(int); ok {
		c.DefaultLimit = limit
	}
}
