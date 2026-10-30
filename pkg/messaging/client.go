package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gTahidi/wabo/internal/request_client"
	"github.com/gTahidi/wabo/manager"
	"github.com/gTahidi/wabo/pkg/graph"
)

// MessagingClient represents a WhatsApp client.
type MessagingClient struct {
	Media             manager.MediaManager
	Message           manager.MessageManager
	PhoneNumberId     string
	ApiAccessToken    string
	BusinessAccountId string
	Requester         *request_client.RequestClient
}

// GetPhoneNumberId returns the phone number ID associated with the client.
func (client *MessagingClient) GetPhoneNumberId() string {
	return client.PhoneNumberId
}

// SetPhoneNumberId sets the phone number ID for the client.
func (client *MessagingClient) SetPhoneNumberId(phoneNumberId string) {
	client.PhoneNumberId = phoneNumberId
}

func (client *MessagingClient) GetApiAccessToken() string {
	return client.ApiAccessToken
}

func (client *MessagingClient) SetApiAccessToken(apiAccessToken string) {
	client.ApiAccessToken = apiAccessToken
}

func (client *MessagingClient) GetBusinessAccountId() string {
	return client.BusinessAccountId
}

type RegisterResponse struct {
	Success bool `json:"success"`
}

// Register registers the phone number for use with the Cloud API. pin is the
// number's six-digit two-step verification PIN; an empty pin is omitted and
// Meta decides whether the request is valid.
func (client *MessagingClient) Register(pin string) (RegisterResponse, error) {
	return client.RegisterContext(context.Background(), pin)
}

// RegisterContext is Register bounded by ctx. A Graph API error is returned
// as a *graph.Error.
func (client *MessagingClient) RegisterContext(ctx context.Context, pin string) (RegisterResponse, error) {
	payload := map[string]string{"messaging_product": "whatsapp"}
	if pin != "" {
		payload["pin"] = pin
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("register: marshalling request body: %w", err)
	}
	return client.postPhoneNumberAction(ctx, "register", string(body))
}

// Deregister removes the phone number from the Cloud API.
func (client *MessagingClient) Deregister() (RegisterResponse, error) {
	return client.DeregisterContext(context.Background())
}

// DeregisterContext is Deregister bounded by ctx. A Graph API error is
// returned as a *graph.Error.
func (client *MessagingClient) DeregisterContext(ctx context.Context) (RegisterResponse, error) {
	return client.postPhoneNumberAction(ctx, "deregister", "")
}

// postPhoneNumberAction POSTs to /{phone_number_id}/{action} and decodes
// Meta's {"success": bool} response.
func (client *MessagingClient) postPhoneNumberAction(ctx context.Context, action, body string) (RegisterResponse, error) {
	if client.Requester == nil {
		return RegisterResponse{}, fmt.Errorf("%s: messaging client has no requester", action)
	}
	if client.PhoneNumberId == "" {
		return RegisterResponse{}, fmt.Errorf("%s: phone number id is required", action)
	}

	apiRequest := client.Requester.NewApiRequest(strings.Join([]string{client.PhoneNumberId, action}, "/"), http.MethodPost)
	if body != "" {
		apiRequest.SetBody(body)
	}
	responseStr, err := apiRequest.ExecuteContext(ctx)
	if err != nil {
		return RegisterResponse{}, err
	}

	var response struct {
		RegisterResponse
		Error *graph.Error `json:"error,omitempty"`
	}
	if err := json.Unmarshal([]byte(responseStr), &response); err != nil {
		return RegisterResponse{}, fmt.Errorf("%s: unmarshalling response: %w", action, err)
	}
	// Graph errors normally arrive with a non-2xx status; this covers one in a 2xx body.
	if response.Error != nil {
		response.Error.HTTPStatus = http.StatusOK
		return response.RegisterResponse, response.Error
	}
	if !response.Success {
		return response.RegisterResponse, fmt.Errorf("%s: graph api returned success=false", action)
	}
	return response.RegisterResponse, nil
}

// SMBSyncType selects the WhatsApp Business app data to sync for a
// coexistence number.
type SMBSyncType string

const (
	// SMBSyncTypeAppState syncs contacts from the WhatsApp Business app.
	SMBSyncTypeAppState SMBSyncType = "SMB_APP_STATE_SYNC"
	// SMBSyncTypeHistory syncs message history from the WhatsApp Business app.
	SMBSyncTypeHistory SMBSyncType = "HISTORY"
)

// SMBAppDataSyncResponse is Meta's response to a sync request.
type SMBAppDataSyncResponse struct {
	MessagingProduct string `json:"messaging_product"`
	RequestId        string `json:"request_id"`
}

// RequestSMBAppDataSyncContext asks Meta to sync WhatsApp Business app data
// for a coexistence number, bounded by ctx. Synced data arrives by webhook.
// A Graph API error is returned as a *graph.Error.
func (client *MessagingClient) RequestSMBAppDataSyncContext(ctx context.Context, syncType SMBSyncType) (SMBAppDataSyncResponse, error) {
	if client.Requester == nil {
		return SMBAppDataSyncResponse{}, errors.New("smb_app_data: messaging client has no requester")
	}
	if client.PhoneNumberId == "" {
		return SMBAppDataSyncResponse{}, errors.New("smb_app_data: phone number id is required")
	}
	if syncType == "" {
		return SMBAppDataSyncResponse{}, errors.New("smb_app_data: sync type is required")
	}
	body, err := json.Marshal(map[string]string{
		"messaging_product": "whatsapp",
		"sync_type":         string(syncType),
	})
	if err != nil {
		return SMBAppDataSyncResponse{}, fmt.Errorf("smb_app_data: marshalling request body: %w", err)
	}

	apiRequest := client.Requester.NewApiRequest(strings.Join([]string{client.PhoneNumberId, "smb_app_data"}, "/"), http.MethodPost)
	apiRequest.SetBody(string(body))
	responseStr, err := apiRequest.ExecuteContext(ctx)
	if err != nil {
		return SMBAppDataSyncResponse{}, err
	}

	var response struct {
		SMBAppDataSyncResponse
		Error *graph.Error `json:"error,omitempty"`
	}
	if err := json.Unmarshal([]byte(responseStr), &response); err != nil {
		return SMBAppDataSyncResponse{}, fmt.Errorf("smb_app_data: unmarshalling response: %w", err)
	}
	// Graph errors normally arrive with a non-2xx status; this covers one in a 2xx body.
	if response.Error != nil {
		response.Error.HTTPStatus = http.StatusOK
		return response.SMBAppDataSyncResponse, response.Error
	}
	return response.SMBAppDataSyncResponse, nil
}
