package events

import "encoding/json"

// UnknownWebhookEvent preserves unrecognized webhook changes, message types,
// and system notifications for application-defined handling.
type UnknownWebhookEvent struct {
	BaseMessageEvent
	Field string          `json:"field"`
	Raw   json.RawMessage `json:"raw"`
}

// GetEventType identifies an unrecognized webhook event.
func (*UnknownWebhookEvent) GetEventType() string { return string(UnknownEventType) }
