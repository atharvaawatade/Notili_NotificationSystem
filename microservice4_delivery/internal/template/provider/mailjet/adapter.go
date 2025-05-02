package mailjet

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	
	mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
	"github.com/mailjet/mailjet-apiv3-go/v4/resources"
	
	"github.com/appointy/notli/microservice4_delivery/internal/template/adapter"
	"github.com/appointy/notli/microservice4_delivery/internal/template/model"
)

const (
	// ProviderName is the unique identifier for the Mailjet template provider
	ProviderName = "mailjet"
)

// Adapter implements the template.Adapter interface for Mailjet
type Adapter struct {
	client *mailjet.Client
	config Config
}

// New creates a new Mailjet template adapter instance
func New(config map[string]interface{}) (adapter.TemplateAdapter, error) {
	var cfg Config
	cfg.FromMap(config)
	
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	
	client := mailjet.NewMailjetClient(cfg.APIKey, cfg.SecretKey)
	
	return &Adapter{
		client: client,
		config: cfg,
	}, nil
}

// Register registers the Mailjet adapter factory with the template registry
func Register(registry *adapter.Registry) {
	registry.Register(ProviderName, New)
}

// GetProviderName returns the provider name
func (a *Adapter) GetProviderName() string {
	return ProviderName
}

// ListTemplates retrieves a list of templates from Mailjet
func (a *Adapter) ListTemplates(ctx context.Context, query *model.TemplateQuery) ([]model.Template, error) {
	// Create base request parameters
	var filters = make(map[string]string)
	filters["OwnerType"] = a.config.TemplateOwnerType
	
	if query != nil {
		if query.Name != "" {
			filters["Name"] = query.Name
		}
		
		if query.Limit > 0 {
			filters["Limit"] = strconv.Itoa(query.Limit)
		} else if a.config.DefaultLimit > 0 {
			filters["Limit"] = strconv.Itoa(a.config.DefaultLimit)
		}
		
		if query.Offset > 0 {
			filters["Offset"] = strconv.Itoa(query.Offset)
		}
	}
	
	// Create a simple request to the template resource
	req := &mailjet.Request{
		Resource: "template",
	}
	
	// Execute the API call
	var result []resources.Template
	err := a.client.Get(req, &result)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAPIRequestFailed, err)
	}
	
	// Convert Mailjet templates to our model
	templates := make([]model.Template, 0, len(result))
	for _, mjTemplate := range result {
		templates = append(templates, a.convertTemplate(&mjTemplate))
	}
	
	return templates, nil
}

// GetTemplate retrieves a single template by ID, including its content
func (a *Adapter) GetTemplate(ctx context.Context, templateID string) (*model.Template, error) {
	// Validate template ID
	id, err := strconv.ParseInt(templateID, 10, 64)
	if err != nil {
		return nil, ErrInvalidTemplateID
	}
	
	// First, get template metadata
	req := &mailjet.Request{
		Resource: "template",
		ID:       id,
	}
	
	var template resources.Template
	err = a.client.Get(req, &template)
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "404") {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrAPIRequestFailed, err)
	}
	
	// Then, get template content
	contentReq := &mailjet.Request{
		Resource: "template",
		ID:       id,
		Action:   "detailcontent",
	}
	
	var content resources.TemplateDetailcontent
	err = a.client.Get(contentReq, &content)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAPIRequestFailed, err)
	}
	
	// Create and populate the template
	result := a.convertTemplate(&template)
	result.HTMLContent = content.HtmlPart
	result.TextContent = content.TextPart
	
	return &result, nil
}

// GetTemplateVariables attempts to extract variables from a template
// Note: Mailjet doesn't directly provide template variables, so we attempt to parse them from content
func (a *Adapter) GetTemplateVariables(ctx context.Context, templateID string) ([]model.TemplateVariable, error) {
	template, err := a.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	
	// Extract variables from content
	variables := extractVariablesFromContent(template.HTMLContent)
	
	return variables, nil
}

// Close releases resources used by the adapter
func (a *Adapter) Close() error {
	// Mailjet client doesn't require explicit cleanup
	return nil
}

// convertTemplate converts a Mailjet template to our model.Template
func (a *Adapter) convertTemplate(mjTemplate *resources.Template) model.Template {
	return model.Template{
		ID:       strconv.FormatInt(mjTemplate.ID, 10),
		Name:     mjTemplate.Name,
		Subject:  "", // Mailjet API doesn't return Subject directly
		Provider: ProviderName,
		Metadata: map[string]interface{}{
			"is_starred": mjTemplate.IsStarred,
			"owner_type": mjTemplate.OwnerType,
			"author":     mjTemplate.Author,
			"owner_id":   mjTemplate.OwnerId,
		},
	}
}

// extractVariablesFromContent extracts variable names from Mailjet template content
// This is a best-effort attempt, as Mailjet doesn't provide direct variable metadata
func extractVariablesFromContent(content string) []model.TemplateVariable {
	// Find variable patterns like {{ var }}, {{var}}, {{ var.property }}, etc.
	// This is a simple implementation - a more robust one would use proper parsing

	// Simple map to deduplicate variables
	variableMap := make(map[string]bool)
	
	// Look for Mailjet variable syntax: {{ varname }}
	// This is a simplified approach and may need refinement
	parts := strings.Split(content, "{{")
	for _, part := range parts[1:] { // Skip the first part (before any variables)
		if closingBrace := strings.Index(part, "}}"); closingBrace != -1 {
			varContent := strings.TrimSpace(part[:closingBrace])
			
			// Skip empty variables and helpers
			if varContent == "" || strings.HasPrefix(varContent, "#") || strings.HasPrefix(varContent, "/") {
				continue
			}
			
			// Extract just the variable name (without any filters or properties)
			varName := strings.Split(varContent, "|")[0]
			varName = strings.Split(varName, ".")[0]
			varName = strings.TrimSpace(varName)
			
			if varName != "" {
				variableMap[varName] = true
			}
		}
	}
	
	// Convert to slice of TemplateVariable
	variables := make([]model.TemplateVariable, 0, len(variableMap))
	for name := range variableMap {
		variables = append(variables, model.TemplateVariable{
			Name:        name,
			Type:        "string", // Default type since we can't determine from content
			Description: "Extracted from template content",
			Required:    false,    // Can't determine if required from content
		})
	}
	
	return variables
}
