package main

import (
	"fmt"
	"os"
	"log"
	"github.com/resend/resend-go/v2"
)

func main() {
	log.Println("Starting direct Resend email test")
	
	// Retrieve API key from environment variable or use the hardcoded one if not set
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		apiKey = "re_j45de3kV_P1tnFP8yTnhsDvRfobCtzxSm"
		log.Println("Using hardcoded API key")
	} else {
		log.Println("Using API key from environment variable")
	}
	
	log.Printf("API Key: %s...", apiKey[:10])

	// Initialize Resend client
	client := resend.NewClient(apiKey)
	log.Println("Initialized Resend client")

	// Prepare email parameters - using exact format as in example
	params := &resend.SendEmailRequest{
		From:    "Simplivu <simplivu@simplivu.com>",
		To:      []string{"atharvaawatade@gmail.com"},
		Subject: "Test Email from Simplivu - Direct Test",
		Html:    "<h1>Hello from Simplivu!</h1><p>This is a direct test email to debug the delivery issues.</p>",
	}

	log.Println("Prepared email parameters:")
	log.Printf("From: %s", params.From)
	log.Printf("To: %v", params.To)
	log.Printf("Subject: %s", params.Subject)

	// Send the email
	log.Println("Sending email...")
	sent, err := client.Emails.Send(params)
	if err != nil {
		log.Printf("ERROR sending email: %v", err)
		// Try to get more details about the error
		log.Printf("Error type: %T", err)
		fmt.Println("Error sending email:", err)
		return
	}
	
	log.Println("Email sent successfully!")
	log.Printf("Message ID: %s", sent.Id)
	fmt.Println("Email sent successfully. ID:", sent.Id)
}
