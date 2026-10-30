package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gTahidi/wabo/internal/request_client"
	"github.com/gTahidi/wabo/pkg/components"
	"github.com/gTahidi/wabo/pkg/graph"
)

// MessageManager is responsible for managing messages.
type MessageManager struct {
	requester     request_client.RequestClient
	PhoneNumberId string
}

// NewMessageManager creates a new instance of MessageManager.
func NewMessageManager(requester request_client.RequestClient, phoneNumberId string) *MessageManager {
	return &MessageManager{
		requester:     requester,
		PhoneNumberId: phoneNumberId,
	}
}

// MessageSendResponse represents the structured API response for sending a message.
type MessageSendResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
	Error *MessageSendError `json:"error,omitempty"`
}

// MessageSendError represents the error object in an API response.
type MessageSendError struct {
	Message   string `json:"message"` // Error description.
	Type      string `json:"type"`    // Error type (e.g., OAuthException).
	Code      int    `json:"code"`    // Error code.
	ErrorData struct {
		MessagingProduct string `json:"messaging_product"`
		Details          string `json:"details"`
	} `json:"error_data"` // Additional error details.
	ErrorSubcode int    `json:"error_subcode"`
	FbtraceID    string `json:"fbtrace_id"`
}

func (e *MessageSendError) graphError(status int) *graph.Error {
	ge := &graph.Error{
		HTTPStatus:   status,
		Message:      e.Message,
		Type:         e.Type,
		Code:         e.Code,
		ErrorSubcode: e.ErrorSubcode,
		FbtraceID:    e.FbtraceID,
	}
	ge.ErrorData.MessagingProduct = e.ErrorData.MessagingProduct
	ge.ErrorData.Details = e.ErrorData.Details
	return ge
}

// StatusResponse represents the API response for status updates (read receipts).
type StatusResponse struct {
	Success bool              `json:"success"`
	Error   *MessageSendError `json:"error,omitempty"`
}

// Reply sends a reply message using the provided BaseMessage and returns a structured response.
// If the API response contains an error, it returns that error.
func (mm *MessageManager) Reply(message components.BaseMessage, phoneNumber string, replyTo string) (*MessageSendResponse, error) {
	return mm.ReplyContext(context.Background(), message, phoneNumber, replyTo)
}

// ReplyContext is Reply bounded by ctx.
func (mm *MessageManager) ReplyContext(ctx context.Context, message components.BaseMessage, phoneNumber string, replyTo string) (*MessageSendResponse, error) {
	return mm.send(ctx, message, components.ApiCompatibleJsonConverterConfigs{
		SendToPhoneNumber: phoneNumber,
		ReplyToMessageId:  replyTo,
	})
}

// Send sends a message using the provided BaseMessage and returns a structured response.
// If the API response contains an error, it returns that error.
func (mm *MessageManager) Send(message components.BaseMessage, phoneNumber string) (*MessageSendResponse, error) {
	return mm.SendContext(context.Background(), message, phoneNumber)
}

// SendContext sends a message, bounded by ctx, and returns Meta's response.
// A Graph API error is returned as a *graph.Error.
func (mm *MessageManager) SendContext(ctx context.Context, message components.BaseMessage, phoneNumber string) (*MessageSendResponse, error) {
	return mm.send(ctx, message, components.ApiCompatibleJsonConverterConfigs{
		SendToPhoneNumber: phoneNumber,
	})
}

func (mm *MessageManager) send(ctx context.Context, message components.BaseMessage, configs components.ApiCompatibleJsonConverterConfigs) (*MessageSendResponse, error) {
	body, err := message.ToJson(configs)
	if err != nil {
		return nil, fmt.Errorf("error converting message to json: %w", err)
	}

	apiRequest := mm.requester.NewApiRequest(strings.Join([]string{mm.PhoneNumberId, "messages"}, "/"), http.MethodPost)
	apiRequest.SetBody(string(body))
	responseStr, err := apiRequest.ExecuteContext(ctx)
	if err != nil {
		return nil, err
	}

	var sendResponse MessageSendResponse
	if err := json.Unmarshal([]byte(responseStr), &sendResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	// Graph errors normally arrive with a non-2xx status; this covers one in a 2xx body.
	if sendResponse.Error != nil {
		return &sendResponse, sendResponse.Error.graphError(http.StatusOK)
	}

	return &sendResponse, nil
}

// MessageID returns the id of the first sent message, or "".
func (r *MessageSendResponse) MessageID() string {
	if r == nil || len(r.Messages) == 0 {
		return ""
	}
	return r.Messages[0].ID
}

// readMessage sends a read receipt for messageId. When showTyping is set, a
// typing indicator is also shown; it is dismissed when you respond or after
// 25 seconds. A Graph API error is returned as a *graph.Error.
func (mm *MessageManager) readMessage(ctx context.Context, messageId string, showTyping bool) error {
	if messageId == "" {
		return fmt.Errorf("error marking message as read: message id is required")
	}

	// Create the request body for marking message as read
	requestBody := map[string]interface{}{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageId,
	}

	// Add typing indicator if requested
	if showTyping {
		requestBody["typing_indicator"] = map[string]string{
			"type": "text",
		}
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error marshalling read request body: %w", err)
	}

	// Build the API request
	apiRequest := mm.requester.NewApiRequest(strings.Join([]string{mm.PhoneNumberId, "messages"}, "/"), http.MethodPost)
	apiRequest.SetBody(string(body))
	responseStr, err := apiRequest.ExecuteContext(ctx)
	if err != nil {
		return fmt.Errorf("error executing read message request: %w", err)
	}

	// Parse the response to check for errors
	var statusResponse StatusResponse
	err = json.Unmarshal([]byte(responseStr), &statusResponse)
	if err != nil {
		return fmt.Errorf("error unmarshalling read response: %w", err)
	}

	// Graph errors normally arrive with a non-2xx status; this covers one in a 2xx body.
	if statusResponse.Error != nil {
		return fmt.Errorf("error marking message as read: %w", statusResponse.Error.graphError(http.StatusOK))
	}
	if !statusResponse.Success {
		return fmt.Errorf("error marking message as read: graph api returned success=false")
	}

	return nil
}

// ReadMessageWithTyping marks a message as read and shows typing indicator.
func (mm *MessageManager) ReadMessageWithTyping(messageId string) error {
	return mm.readMessage(context.Background(), messageId, true)
}

// ReadMessageWithTypingContext is ReadMessageWithTyping bounded by ctx.
func (mm *MessageManager) ReadMessageWithTypingContext(ctx context.Context, messageId string) error {
	return mm.readMessage(ctx, messageId, true)
}

// ReadMessageOnly marks a message as read without showing typing indicator.
func (mm *MessageManager) ReadMessageOnly(messageId string) error {
	return mm.readMessage(context.Background(), messageId, false)
}

// ReadMessageOnlyContext is ReadMessageOnly bounded by ctx.
func (mm *MessageManager) ReadMessageOnlyContext(ctx context.Context, messageId string) error {
	return mm.readMessage(ctx, messageId, false)
}
