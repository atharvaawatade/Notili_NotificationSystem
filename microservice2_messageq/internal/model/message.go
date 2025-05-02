package model

// EmailChannelDTO represents email-specific payload
// Matches validation rules from docs
// Note: CC/BCC arrays limited to 50 addresses each in validator tags

type EmailChannelDTO struct {
    Subject string   `json:"subject" validate:"required,min=1,max=998"`
    CC      []string `json:"cc" validate:"omitempty,max=50,dive,email"`
    BCC     []string `json:"bcc" validate:"omitempty,max=50,dive,email"`
}

type NotificationMessage struct {
    IdempotencyKey string      `json:"idempotency_key"`
    Recipient      string      `json:"recipient"`
    TemplateID     string      `json:"template_id"`
    Priority       string      `json:"priority"`
    MessageType    string      `json:"message_type"`
    Channel        string      `json:"channel"`
    ChannelData    interface{} `json:"channel_data"`
}
