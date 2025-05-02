package mailjet

import "errors"

// Error definitions specific to the Mailjet adapter
var (
	ErrMissingCredentials     = errors.New("missing Mailjet API credentials")
	ErrInvalidTemplateID      = errors.New("invalid Mailjet template ID")
	ErrAPIRequestFailed       = errors.New("Mailjet API request failed")
	ErrFailedToParseMJResponse = errors.New("failed to parse Mailjet API response")
	ErrTemplateNotFound       = errors.New("template not found in Mailjet")
)
