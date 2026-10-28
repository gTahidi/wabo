package wapi

import (
	"net/http"

	"github.com/gTahidi/wapi.go/internal/request_client"
	"github.com/gTahidi/wapi.go/manager"
	"github.com/gTahidi/wapi.go/pkg/business"
	"github.com/gTahidi/wapi.go/pkg/events"
	"github.com/gTahidi/wapi.go/pkg/messaging"
	"github.com/labstack/echo/v4"
)

type ClientConfig struct {
	BusinessAccountId string
	ApiAccessToken    string
	WebhookSecret     string `validate:"required"`

	// these two are not required, because may be user want to use their own server
	WebhookPath       string
	WebhookServerPort int

	// AppSecret, when set, makes the webhook POST handler reject payloads
	// whose X-Hub-Signature-256 does not match (HMAC-SHA256 of the raw body).
	AppSecret string

	// HTTPClient is used for Graph API calls; nil uses a shared client with
	// a 30 second timeout. Per-call deadlines come from the *Context methods.
	HTTPClient *http.Client
}

type Client struct {
	Business     business.BusinessClient     // Business is the business client.
	Messaging    []messaging.MessagingClient // MessagingClient is the messaging client.
	eventManager *manager.EventManager       // eventManager is the event manager.
	webhook      *manager.WebhookManager     // webhook is the webhook manager.
	requester    *request_client.RequestClient

	apiAccessToken    string
	businessAccountId string
}

func New(config *ClientConfig) *Client {
	eventManager := manager.NewEventManager()
	requester := *request_client.NewRequestClientWithHTTPClient(config.ApiAccessToken, config.HTTPClient)
	return &Client{
		businessAccountId: config.BusinessAccountId,
		apiAccessToken:    config.ApiAccessToken,
		Messaging:         []messaging.MessagingClient{},
		eventManager:      eventManager,
		Business: *business.NewBusinessClient(&business.BusinessClientConfig{
			BusinessAccountId: config.BusinessAccountId,
			AccessToken:       config.ApiAccessToken,
			Requester:         &requester,
		}),
		webhook:   manager.NewWebhook(&manager.WebhookManagerConfig{Path: config.WebhookPath, Secret: config.WebhookSecret, AppSecret: config.AppSecret, Port: config.WebhookServerPort, EventManager: eventManager, Requester: requester}),
		requester: &requester,
	}
}

func (client *Client) NewMessagingClient(phoneNumberId string) *messaging.MessagingClient {
	// Create a new request client

	// Create a new Client instance with the provided configurations
	messagingClient := &messaging.MessagingClient{
		Media:             *manager.NewMediaManager(*client.requester),
		Message:           *manager.NewMessageManager(*client.requester, phoneNumberId),
		PhoneNumberId:     phoneNumberId,
		ApiAccessToken:    client.apiAccessToken,
		BusinessAccountId: client.businessAccountId,
		Requester:         client.requester,
	}

	client.Messaging = append(client.Messaging, *messagingClient)
	return messagingClient
}

// GetWebhookGetRequestHandler returns the handler function for handling GET requests to the webhook.
func (client *Client) GetWebhookGetRequestHandler() func(c echo.Context) error {
	return client.webhook.GetRequestHandler
}

// GetWebhookPostRequestHandler returns the handler function for handling POST requests to the webhook.
func (client *Client) GetWebhookPostRequestHandler() func(c echo.Context) error {
	return client.webhook.PostRequestHandler
}

// OnMessage registers a handler for a specific event type.
func (client *Client) On(eventType events.EventType, handler func(events.BaseEvent)) {
	client.webhook.
		EventManager.On(eventType, handler)
}

// InitiateClient initializes the client and starts listening to events from the webhook.
// It returns true if the client was successfully initiated.
func (client *Client) Initiate() bool {
	client.webhook.ListenToEvents()
	return true
}

// GetMediaManager returns a MediaManager for uploading media (e.g., for template headers).
// This is useful when you need to upload media without a specific phone number context.
func (client *Client) GetMediaManager() *manager.MediaManager {
	return manager.NewMediaManager(*client.requester)
}

// GetFlowManager returns a FlowManager for WhatsApp Flows management operations.
// This includes creating, listing, updating, publishing, and deprecating flows.
func (client *Client) GetFlowManager() *manager.FlowManager {
	return manager.NewFlowManager(&manager.FlowManagerConfig{
		BusinessAccountId: client.businessAccountId,
		ApiAccessToken:    client.apiAccessToken,
		Requester:         client.requester,
	})
}
