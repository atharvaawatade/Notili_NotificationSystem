package template

import (
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/appointy/notli/microservice4_delivery/internal/template/provider/mailjet"
)

// SetupOptions contains options for setting up the template service
type SetupOptions struct {
	// Mailjet options
	MailjetAPIKey    string
	MailjetSecretKey string
}

// DefaultSetupOptions creates options with values from environment variables
func DefaultSetupOptions() *SetupOptions {
	return &SetupOptions{
		MailjetAPIKey:    os.Getenv("MAILJET_API_KEY"),
		MailjetSecretKey: os.Getenv("MAILJET_SECRET_KEY"),
	}
}

// SetupTemplateEngine initializes a template engine suitable for the application needs
func SetupTemplateEngine(options *SetupOptions) (TemplateEngineInterface, error) {
	// Create default in-house template engine for transactional emails
	defaultEngine := NewEngine(EngineOptions{
		TemplateDirs:   []string{"templates"},
		ReloadInterval: 5 * time.Minute,
		// Using nil for FunctionsMap will trigger the default function map in NewEngine
		FunctionsMap:   nil,
	})
	
	// Create function map for the templates
	funcMap := defaultFunctionMap()
	
	// Add our pre-designed transaction email template
	transactionSubject := template.Must(template.New("subject").Funcs(funcMap).Parse("{{.AppName}} - {{.Subject}}"))
	
	transactionPlain := template.Must(template.New("plain").Funcs(funcMap).Parse(`Hi {{.Name}},

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

	// Green and white themed HTML template
	transactionHtml := template.Must(template.New("html").Funcs(funcMap).Parse(`<!DOCTYPE html>
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
	
	// Register the transaction template
	defaultEngine.AddTemplate(&Template{
		ID:            "transaction",
		Name:          "Transactional Email",
		Description:   "Standard template for transactional emails with green/white theme",
		SubjectTmpl:   transactionSubject,
		PlainTextTmpl: transactionPlain,
		HtmlTmpl:      transactionHtml,
		LastModified:  time.Now(),
	})

	// For promotional emails - we'll use Mailjet in the future, but for now it's commented out
	/* 
	if options.MailjetAPIKey != "" && options.MailjetSecretKey != "" {
		// Initialize and return Mailjet template engine
		return NewMailjetTemplateEngine(options.MailjetAPIKey, options.MailjetSecretKey)
	}
	*/

	return defaultEngine, nil
}

// Setup initializes the template service with the provided options
func Setup(options *SetupOptions) (*Service, error) {
	if options == nil {
		options = DefaultSetupOptions()
	}

	// Create configuration
	cfg := &Config{
		DefaultProvider: mailjet.ProviderName,
		Providers: map[string]map[string]interface{}{},
	}

	// Add Mailjet provider if configured (for future promotional emails)
	if options.MailjetAPIKey != "" && options.MailjetSecretKey != "" {
		cfg.Providers[mailjet.ProviderName] = map[string]interface{}{
			"api_key":             options.MailjetAPIKey,
			"secret_key":          options.MailjetSecretKey,
			"template_owner_type": "user",
			"default_limit":       100,
		}
	} else {
		// Instead of returning an error, just log that we're using in-house templates
		fmt.Println("No third-party template providers configured. Using in-house templates for all emails.")
	}

	// Create and return the service
	return NewService(cfg)
}
