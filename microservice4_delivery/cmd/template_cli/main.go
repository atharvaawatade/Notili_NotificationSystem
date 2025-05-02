package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/appointy/notli/microservice4_delivery/internal/template"
	"github.com/appointy/notli/microservice4_delivery/internal/template/model"
)

const (
	// Mailjet credentials
	defaultMailjetAPIKey    = "cc35d2d982f49df106a504265ca30e7b"
	defaultMailjetSecretKey = "771a11942eeac0bbfda1076dbac4c8ed"
)

func main() {
	// Define command-line flags
	apiKey := flag.String("api-key", defaultMailjetAPIKey, "Mailjet API key")
	secretKey := flag.String("secret-key", defaultMailjetSecretKey, "Mailjet Secret key")
	listCmd := flag.Bool("list", false, "List available templates")
	getCmd := flag.String("get", "", "Get template by ID")
	varsCmd := flag.String("vars", "", "Get variables for template by ID")
	flag.Parse()

	// Initialize template service
	options := &template.SetupOptions{
		MailjetAPIKey:    *apiKey,
		MailjetSecretKey: *secretKey,
	}

	svc, err := template.Setup(options)
	if err != nil {
		fmt.Printf("Error initializing template service: %v\n", err)
		os.Exit(1)
	}
	defer svc.Close()

	ctx := context.Background()

	// Execute requested command
	if *listCmd {
		// List templates
		templates, err := svc.ListTemplates(ctx, "mailjet", &model.TemplateQuery{
			Limit: 10, // Limit to 10 templates for demonstration
		})
		if err != nil {
			fmt.Printf("Error listing templates: %v\n", err)
			os.Exit(1)
		}

		// Print templates in a nicely formatted table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tName\tSubject\tProvider")
		fmt.Fprintln(w, "-------------------------------------------------")
		for _, t := range templates {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.ID, t.Name, t.Subject, t.Provider)
		}
		w.Flush()
		fmt.Printf("\nFound %d templates\n", len(templates))

	} else if *getCmd != "" {
		// Get template by ID
		templateID := *getCmd
		t, err := svc.GetTemplate(ctx, "mailjet", templateID)
		if err != nil {
			fmt.Printf("Error getting template %s: %v\n", templateID, err)
			os.Exit(1)
		}

		// Print template details
		fmt.Printf("Template ID: %s\n", t.ID)
		fmt.Printf("Name: %s\n", t.Name)
		fmt.Printf("Subject: %s\n", t.Subject)
		fmt.Printf("Provider: %s\n", t.Provider)
		
		// Print content snippets
		htmlPreview := t.HTMLContent
		if len(htmlPreview) > 100 {
			htmlPreview = htmlPreview[:100] + "..."
		}
		fmt.Printf("HTML Content (preview): %s\n", htmlPreview)
		
		textPreview := t.TextContent
		if len(textPreview) > 100 {
			textPreview = textPreview[:100] + "..."
		}
		fmt.Printf("Text Content (preview): %s\n", textPreview)

	} else if *varsCmd != "" {
		// Get template variables
		templateID := *varsCmd
		vars, err := svc.GetTemplateVariables(ctx, "mailjet", templateID)
		if err != nil {
			fmt.Printf("Error getting variables for template %s: %v\n", templateID, err)
			os.Exit(1)
		}

		// Print variables
		fmt.Printf("Variables for template %s:\n", templateID)
		fmt.Println("-------------------------------------------------")
		for i, v := range vars {
			required := ""
			if v.Required {
				required = " (required)"
			}
			fmt.Printf("%d. %s (type: %s)%s\n", i+1, v.Name, v.Type, required)
			if v.Description != "" && !strings.Contains(v.Description, "Extracted from") {
				fmt.Printf("   Description: %s\n", v.Description)
			}
		}
		fmt.Printf("\nFound %d variables\n", len(vars))

	} else {
		fmt.Println("No command specified. Use --list, --get=ID, or --vars=ID")
		flag.Usage()
		os.Exit(1)
	}
}
