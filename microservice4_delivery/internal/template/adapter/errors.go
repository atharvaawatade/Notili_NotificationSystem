package adapter

import "errors"

// Error definitions for template adapters
var (
	ErrProviderNotFound      = errors.New("template provider not found")
	ErrInvalidConfiguration  = errors.New("invalid template provider configuration")
	ErrTemplateNotFound      = errors.New("template not found")
	ErrFailedToFetchTemplate = errors.New("failed to fetch template")
	ErrAuthenticationFailed  = errors.New("template provider authentication failed")
	ErrInvalidTemplateID     = errors.New("invalid template ID")
)
