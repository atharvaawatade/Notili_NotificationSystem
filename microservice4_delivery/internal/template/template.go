package template

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"sync"
	"time"
)

// TemplateEngineInterface defines the interface for template operations
type TemplateEngineInterface interface {
	Render(templateID string, data map[string]interface{}) (string, string, string, error)
	Get(templateID string) (*Template, error)
	ReloadTemplates() error
	AddTemplate(tmpl *Template)
}

// Engine implements the TemplateEngineInterface for template rendering and management
type Engine struct {
	templates       map[string]*Template
	templateDirs    []string
	reloadInterval  time.Duration
	lastReloadTime  time.Time
	functionsMap    template.FuncMap
	mu              sync.RWMutex
}

// Template represents a single email template with subject, plaintext and HTML versions
type Template struct {
	ID             string
	Name           string
	Description    string
	SubjectTmpl    *template.Template
	PlainTextTmpl  *template.Template
	HtmlTmpl       *template.Template
	LastModified   time.Time
}

// EngineOptions contains configuration options for the template engine
type EngineOptions struct {
	TemplateDirs    []string
	ReloadInterval  time.Duration
	FunctionsMap    template.FuncMap
}

// NewEngine creates a new template engine
func NewEngine(opts EngineOptions) *Engine {
	// Set default function map if not provided
	funcMap := opts.FunctionsMap
	if funcMap == nil {
		funcMap = defaultFunctionMap()
	}

	engine := &Engine{
		templates:      make(map[string]*Template),
		templateDirs:   opts.TemplateDirs,
		reloadInterval: opts.ReloadInterval,
		functionsMap:   funcMap,
	}

	// Initial load of templates
	_ = engine.ReloadTemplates()

	return engine
}

// defaultFunctionMap provides a set of useful template functions
func defaultFunctionMap() template.FuncMap {
	return template.FuncMap{
		"formatDate": func(t time.Time, layout string) string {
			return t.Format(layout)
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"join": func(sep string, items []string) string {
			return strings.Join(items, sep)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"formatCurrency": func(amount float64, currency string) string {
			switch currency {
			case "USD":
				return fmt.Sprintf("$%.2f", amount)
			case "EUR":
				return fmt.Sprintf("€%.2f", amount)
			case "GBP":
				return fmt.Sprintf("£%.2f", amount)
			case "INR":
				return fmt.Sprintf("₹%.2f", amount)
			default:
				return fmt.Sprintf("%.2f %s", amount, currency)
			}
		},
	}
}

// Get retrieves a template by its ID
func (e *Engine) Get(templateID string) (*Template, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check if it's time to reload templates
	if e.reloadInterval > 0 && time.Since(e.lastReloadTime) > e.reloadInterval {
		e.mu.RUnlock()
		if err := e.ReloadTemplates(); err != nil {
			return nil, err
		}
		e.mu.RLock()
	}

	tmpl, exists := e.templates[templateID]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	return tmpl, nil
}

// ReloadTemplates reloads all templates from disk
func (e *Engine) ReloadTemplates() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// TODO: Implement file-based template loading logic
	// For now, we'll use some hard-coded templates for demonstration

	// Add transactional email template with green and white theme
	transactionSubject := template.Must(template.New("subject").Funcs(e.functionsMap).Parse("{{.AppName}} - {{.Subject}}"))
	
	transactionPlain := template.Must(template.New("plain").Funcs(e.functionsMap).Parse(`Hi {{.Name}},

{{.Message}}

{{if .ActionText}}{{.ActionText}}: {{.ActionURL}}{{end}}

{{if .Details}}
Details:
{{range $key, $value := .Details}}
{{$key}}: {{$value}}
{{end}}
{{end}}

If you have any questions, please contact our support team.

Best regards,
The {{.AppName}} Team`))

	// Create a beautiful and professional HTML template with green and white theme
	transactionHtml := template.Must(template.New("html").Funcs(e.functionsMap).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Subject}}</title>
    <style>
        /* Base styles */
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333333;
            background-color: #f9f9f9;
            margin: 0;
            padding: 0;
        }
        .email-container {
            max-width: 600px;
            margin: 0 auto;
            background-color: #ffffff;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 4px 10px rgba(0, 0, 0, 0.05);
        }
        .email-header {
            background-color: #10b981; /* Green header */
            padding: 24px;
            text-align: center;
        }
        .email-header img {
            max-height: 40px;
        }
        .email-body {
            padding: 32px 24px;
            background-color: #ffffff;
        }
        .email-footer {
            background-color: #f3f4f6;
            color: #6b7280;
            font-size: 14px;
            text-align: center;
            padding: 16px 24px;
            border-top: 1px solid #e5e7eb;
        }
        h1 {
            color: #10b981;
            font-weight: 600;
            margin-top: 0;
            margin-bottom: 24px;
            font-size: 24px;
        }
        p {
            margin: 0 0 16px;
        }
        .message {
            margin-bottom: 24px;
        }
        .button {
            display: inline-block;
            background-color: #10b981;
            color: #ffffff !important;
            text-decoration: none;
            padding: 12px 24px;
            border-radius: 4px;
            font-weight: 500;
            margin: 16px 0;
            text-align: center;
            transition: background-color 0.2s;
        }
        .button:hover {
            background-color: #059669;
        }
        .details-table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
            font-size: 15px;
        }
        .details-table th, .details-table td {
            padding: 12px 15px;
            border: 1px solid #e5e7eb;
            text-align: left;
        }
        .details-table th {
            background-color: #f3f4f6;
            font-weight: 600;
            color: #374151;
        }
        .details-table tr:nth-child(even) {
            background-color: #f9fafb;
        }
        .details-table tr:hover {
            background-color: #f3f4f6;
        }
        @media only screen and (max-width: 620px) {
            .email-container {
                width: 100% !important;
            }
            .email-body, .email-header, .email-footer {
                padding: 20px 15px !important;
            }
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="email-header">
            <img src="{{.LogoURL}}" alt="{{.AppName}} Logo" onerror="this.style.display='none'">
            {{if not .LogoURL}}<h2 style="color: #ffffff; margin: 0;">{{.AppName}}</h2>{{end}}
        </div>
        <div class="email-body">
            <h1>{{.Subject}}</h1>
            <div class="message">
                <p>Hi {{.Name}},</p>
                <p>{{.Message}}</p>
            </div>
            {{if .ActionText}}
            <a href="{{.ActionURL}}" class="button">{{.ActionText}}</a>
            {{end}}
            {{if .Details}}
            <table class="details-table">
                <thead>
                    <tr>
                        <th>Item</th>
                        <th>Details</th>
                    </tr>
                </thead>
                <tbody>
                    {{range $key, $value := .Details}}
                    <tr>
                        <td>{{$key}}</td>
                        <td>{{$value}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
            {{end}}
            <p>If you have any questions, please don't hesitate to contact our support team.</p>
            <p>Best regards,<br>The {{.AppName}} Team</p>
        </div>
        <div class="email-footer">
            <p>&copy; {{formatDate .Now "2006"}} {{.AppName}}. All rights reserved.</p>
            {{if .UnsubscribeURL}}<p><a href="{{.UnsubscribeURL}}" style="color: #6b7280;">Unsubscribe</a></p>{{end}}
        </div>
    </div>
</body>
</html>`))
	
	e.templates["transaction"] = &Template{
		ID:            "transaction",
		Name:          "Transactional Email",
		Description:   "Standard template for transactional emails like confirmations, notifications, etc.",
		SubjectTmpl:   transactionSubject,
		PlainTextTmpl: transactionPlain,
		HtmlTmpl:      transactionHtml,
		LastModified:  time.Now(),
	}

	// Add some example templates
	welcomeSubject := template.Must(template.New("subject").Funcs(e.functionsMap).Parse("Welcome to {{.AppName}}, {{.Name}}!"))
	welcomePlain := template.Must(template.New("plain").Funcs(e.functionsMap).Parse("Hi {{.Name}},\n\nWelcome to {{.AppName}}! We're excited to have you on board.\n\nThe {{.AppName}} Team"))
	welcomeHtml := template.Must(template.New("html").Funcs(e.functionsMap).Parse("<h1>Welcome, {{.Name}}!</h1><p>We're excited to have you on board at {{.AppName}}.</p><p>The {{.AppName}} Team</p>"))
	
	e.templates["welcome"] = &Template{
		ID:            "welcome",
		Name:          "Welcome Email",
		Description:   "Sent to new users upon registration",
		SubjectTmpl:   welcomeSubject,
		PlainTextTmpl: welcomePlain,
		HtmlTmpl:      welcomeHtml,
		LastModified:  time.Now(),
	}

	passwordResetSubject := template.Must(template.New("subject").Funcs(e.functionsMap).Parse("Password Reset for {{.AppName}}"))
	passwordResetPlain := template.Must(template.New("plain").Funcs(e.functionsMap).Parse("Hi {{.Name}},\n\nYou recently requested to reset your password for your {{.AppName}} account. Click the link below to reset it.\n\n{{.ResetLink}}\n\nIf you did not request a password reset, please ignore this email.\n\nThe {{.AppName}} Team"))
	passwordResetHtml := template.Must(template.New("html").Funcs(e.functionsMap).Parse("<h1>Password Reset</h1><p>Hi {{.Name}},</p><p>You recently requested to reset your password for your {{.AppName}} account. Click the button below to reset it.</p><p><a href=\"{{.ResetLink}}\" style=\"background-color: #4CAF50; color: white; padding: 10px 15px; text-decoration: none; border-radius: 4px;\">Reset Password</a></p><p>If you did not request a password reset, please ignore this email.</p><p>The {{.AppName}} Team</p>"))
	
	e.templates["password_reset"] = &Template{
		ID:            "password_reset",
		Name:          "Password Reset",
		Description:   "Sent when a user requests a password reset",
		SubjectTmpl:   passwordResetSubject,
		PlainTextTmpl: passwordResetPlain,
		HtmlTmpl:      passwordResetHtml,
		LastModified:  time.Now(),
	}

	e.lastReloadTime = time.Now()
	return nil
}

// AddTemplate adds or updates a template in memory
func (e *Engine) AddTemplate(tmpl *Template) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.templates[tmpl.ID] = tmpl
}

// Render renders the subject, plaintext and HTML versions of a template with the given data
func (e *Engine) Render(templateID string, data map[string]interface{}) (subject, plainText, html string, err error) {
	tmpl, err := e.Get(templateID)
	if err != nil {
		return "", "", "", err
	}

	// Render subject
	var subjectBuf bytes.Buffer
	if err = tmpl.SubjectTmpl.Execute(&subjectBuf, data); err != nil {
		return "", "", "", fmt.Errorf("error rendering subject: %w", err)
	}
	subject = subjectBuf.String()

	// Render plain text
	var plainBuf bytes.Buffer
	if err = tmpl.PlainTextTmpl.Execute(&plainBuf, data); err != nil {
		return "", "", "", fmt.Errorf("error rendering plain text: %w", err)
	}
	plainText = plainBuf.String()

	// Render HTML
	var htmlBuf bytes.Buffer
	if err = tmpl.HtmlTmpl.Execute(&htmlBuf, data); err != nil {
		return "", "", "", fmt.Errorf("error rendering HTML: %w", err)
	}
	html = htmlBuf.String()

	return subject, plainText, html, nil
}
