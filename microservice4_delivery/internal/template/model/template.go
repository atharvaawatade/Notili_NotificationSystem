package model

// Template represents an email template with its content and metadata
type Template struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Subject     string                 `json:"subject"`
	HTMLContent string                 `json:"html_content"`
	TextContent string                 `json:"text_content"`
	Provider    string                 `json:"provider"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"` // Provider-specific data
}

// TemplateQuery represents query parameters for filtering templates
type TemplateQuery struct {
	Name   string `json:"name,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	// Add more filter options as needed
}

// TemplateVariable represents a variable that can be used in a template
type TemplateVariable struct {
	Name        string `json:"name"`
	Type        string `json:"type"`        // string, number, boolean, object, array
	Description string `json:"description"` // Optional description
	Required    bool   `json:"required"`    // Whether this variable is required
}
