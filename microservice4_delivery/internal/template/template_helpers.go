package template

import (
	"html/template"
)

// ParseText is a helper function that parses a string into a template
func ParseText(name, content string) (*template.Template, error) {
	return template.New(name).Parse(content)
}
