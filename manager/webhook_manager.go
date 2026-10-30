package manager

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gTahidi/wabo/internal"
	"github.com/gTahidi/wabo/internal/request_client"
	"github.com/gTahidi/wabo/pkg/components"
	"github.com/gTahidi/wabo/pkg/events"
	"github.com/labstack/echo/v4"
)

// WebhookManager represents a manager for handling webhooks.
type WebhookManager struct {
	secret       string
	appSecret    string
	path         string
	port         int
	EventManager *EventManager
	Requester    request_client.RequestClient
}

// WebhookManagerConfig represents the configuration options for creating a new WebhookManager.
type WebhookManagerConfig struct {
	Secret string `validate:"required"`
	// AppSecret, when set, makes PostRequestHandler reject payloads whose
	// X-Hub-Signature-256 header does not match.
	AppSecret    string
	EventManager *EventManager                `validate:"required"`
	Requester    request_client.RequestClient `validate:"required"`
	Path         string
	Port         int
}

// NewWebhook creates a new WebhookManager with the given options.
func NewWebhook(options *WebhookManagerConfig) *WebhookManager {
	if err := internal.GetValidator().Struct(options); err != nil {
		return nil
	}
	return &WebhookManager{
		secret:       options.Secret,
		appSecret:    options.AppSecret,
		path:         options.Path,
		port:         options.Port,
		EventManager: options.EventManager,
		Requester:    options.Requester,
	}
}

// createEchoHttpServer creates a new instance of Echo HTTP server.
// This function is used in case the client has not provided any custom HTTP server.
func (wh *WebhookManager) createEchoHttpServer() *echo.Echo {
	e := echo.New()
	return e
}

// GetRequestHandler handles GET requests to the webhook endpoint: Meta's
// verification request, answered with hub.challenge when hub.verify_token
// matches the configured secret.
func (wh *WebhookManager) GetRequestHandler(c echo.Context) error {
	hubVerificationToken := c.QueryParam("hub.verify_token")
	hubChallenge := c.QueryParam("hub.challenge")
	hubMode := c.QueryParam("hub.mode")
	if hubMode == "subscribe" && wh.secret != "" &&
		subtle.ConstantTimeCompare([]byte(hubVerificationToken), []byte(wh.secret)) == 1 {
		return c.String(http.StatusOK, hubChallenge)
	}
	return c.String(http.StatusBadRequest, "invalid token")
}

// VerifyWebhookSignature reports whether signatureHeader (the
// X-Hub-Signature-256 header, "sha256=<hex>") is the HMAC-SHA256 of rawBody
// keyed with the Meta app secret. The comparison is constant-time.
func VerifyWebhookSignature(rawBody []byte, signatureHeader, appSecret string) bool {
	const prefix = "sha256="
	if appSecret == "" || !strings.HasPrefix(signatureHeader, prefix) {
		return false
	}
	expected, err := hex.DecodeString(strings.TrimPrefix(signatureHeader, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(rawBody)
	return hmac.Equal(mac.Sum(nil), expected)
}

// PostRequestHandler handles POST requests to the webhook endpoint. When an
// app secret is configured, payloads with a missing or invalid
// X-Hub-Signature-256 are rejected with 401.
func (wh *WebhookManager) PostRequestHandler(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusBadRequest, "error reading request body")
	}

	if wh.appSecret != "" && !VerifyWebhookSignature(body, c.Request().Header.Get("X-Hub-Signature-256"), wh.appSecret) {
		return c.String(http.StatusUnauthorized, "invalid signature")
	}

	var payload WhatsappApiNotificationPayloadSchemaType
	if err := json.Unmarshal(body, &payload); err != nil {
		return c.String(http.StatusBadRequest, "Invalid JSON data")
	}

	if err := internal.GetValidator().Struct(payload); err != nil {
		return c.String(http.StatusBadRequest, "Invalid JSON data")
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			switch change.Field {
			case WebhookFieldEnumMessages:
				var messageValue MessagesValue
				valueBytes, err := json.Marshal(change.Value)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error marshaling messages value")
				}
				if err := json.Unmarshal(valueBytes, &messageValue); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid MessagesValue JSON: %v", err))
				}

				err = wh.handleMessagesSubscriptionEvents(HandleMessageSubscriptionEventPayload{
					Messages: messageValue.Messages,
					Statuses: messageValue.Statuses,
					PhoneNumber: events.BusinessPhoneNumber{
						DisplayNumber: messageValue.Metadata.DisplayPhoneNumber,
						Id:            messageValue.Metadata.PhoneNumberId,
					},
					BusinessAccountId: entry.Id,
					Contacts:          messageValue.Contacts,
					UserActions:       messageValue.UserActions,
				})

				if err != nil {
					c.String(http.StatusInternalServerError, "Internal server error")
					return err
				}
			case WebhookFieldEnumAccountReview:
				var accountReviewValue AccountReviewUpdateValue
				valueBytes, err := json.Marshal(change.Value)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error marshaling messages value")
				}
				if err := json.Unmarshal(valueBytes, &accountReviewValue); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid accountReviewValue JSON: %v", err))
				}
				err = wh.handleAccountReviewSubscriptionEvents(events.BaseBusinessAccountEvent{
					BusinessAccountId: entry.Id,
					Timestamp:         entryTime(entry.Time),
				}, accountReviewValue)
				if err != nil {
					c.String(http.StatusInternalServerError, "Internal server error")
					return err
				}
			case WebhookFieldEnumAccountAlerts:
				var accountAlertValue AccountAlertsValue
				valueBytes, err := json.Marshal(change.Value)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error marshaling messages value")
				}
				if err := json.Unmarshal(valueBytes, &accountAlertValue); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid accountAlertValue JSON: %v", err))
				}
				err = wh.handleAccountAlertsSubscriptionEvents(events.BaseBusinessAccountEvent{
					BusinessAccountId: entry.Id,
					Timestamp:         entryTime(entry.Time),
				}, accountAlertValue)
				if err != nil {
					c.String(http.StatusInternalServerError, "Internal server error")
					return err
				}
			case WebhookFieldEnumAccountUpdate:
				var accountUpdate AccountUpdateValue
				valueBytes, err := json.Marshal(change.Value)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error marshaling messages value")
				}
				if err := json.Unmarshal(valueBytes, &accountUpdate); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid accountUpdate JSON: %v", err))
				}
				wh.handleAccountUpdateSubscriptionEvents(events.BaseBusinessAccountEvent{
					BusinessAccountId: entry.Id,
					Timestamp:         entryTime(entry.Time),
				}, accountUpdate)
				if err != nil {
					c.String(http.StatusInternalServerError, "Internal server error")
					return err
				}
			case WebhookFieldEnumTemplateCategoryUpdate:
				var templateCategoryUpdate TemplateCategoryUpdateValue
				valueBytes, err := json.Marshal(change.Value)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error marshaling messages value")
				}
				if err := json.Unmarshal(valueBytes, &templateCategoryUpdate); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid templateCategoryUpdate JSON: %v", err))
				}
				wh.handleTemplateCategoryUpdateSubscriptionEvents(events.BaseBusinessAccountEvent{
					BusinessAccountId: entry.Id,
					Timestamp:         entryTime(entry.Time),
				}, templateCategoryUpdate)
				if err != nil {
					c.String(http.StatusInternalServerError, "Internal server error")
					return err
				}
			case WebhookFieldEnumMessageTemplateQuality:
				var qualityUpdate TemplateQualityUpdateValue
				valueBytes, err := json.Marshal(change.Value)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error marshaling messages value")
				}
				if err := json.Unmarshal(valueBytes, &qualityUpdate); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid qualityUpdate JSON: %v", err))
				}
				wh.handleMessageTemplateQualitySubscriptionEvents(events.BaseBusinessAccountEvent{
					BusinessAccountId: entry.Id,
					Timestamp:         entryTime(entry.Time),
				}, qualityUpdate)
				if err != nil {
					c.String(http.StatusInternalServerError, "Internal server error")
					return err
				}
			case WebhookFieldEnumMessageTemplateStatus:
				var statusUpdate TemplateStatusUpdateValue
				valueBytes, err := json.Marshal(change.Value)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error marshaling messages value")
				}
				if err := json.Unmarshal(valueBytes, &statusUpdate); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid statusUpdate JSON: %v", err))
				}
				wh.handleMessageTemplateStatusSubscriptionEvents(events.BaseBusinessAccountEvent{
					BusinessAccountId: entry.Id,
					Timestamp:         entryTime(entry.Time),
				}, statusUpdate)
				if err != nil {
					c.String(http.StatusInternalServerError, "Internal server error")
					return err
				}
			case WebhookFieldEnumPhoneNumberName:
				var nameUpdate PhoneNumberNameUpdateValue
				valueBytes, err := json.Marshal(change.Value)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error marshaling messages value")
				}
				if err := json.Unmarshal(valueBytes, &nameUpdate); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid nameUpdate JSON: %v", err))
				}
				wh.handlePhoneNumberNameSubscriptionEvents(events.BaseBusinessAccountEvent{
					BusinessAccountId: entry.Id,
					Timestamp:         entryTime(entry.Time),
				}, nameUpdate)
				if err != nil {
					c.String(http.StatusInternalServerError, "Internal server error")
					return err
				}
			case WebhookFieldEnumPhoneNumberQuality:
				var qualityUpdate PhoneNumberQualityUpdateValue
				valueBytes, err := json.Marshal(change.Value)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error marshaling messages value")
				}
				if err := json.Unmarshal(valueBytes, &qualityUpdate); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid qualityUpdate JSON: %v", err))
				}
				wh.handlePhoneNumberQualitySubscriptionEvents(events.BaseBusinessAccountEvent{
					BusinessAccountId: entry.Id,
					Timestamp:         entryTime(entry.Time),
				}, qualityUpdate)
				if err != nil {
					c.String(http.StatusInternalServerError, "Internal server error")
					return err
				}
			case WebhookFieldEnumBusinessCapability:
				var capabilityUpdate BusinessCapabilityUpdateValue
				valueBytes, err := json.Marshal(change.Value)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error marshaling messages value")
				}
				if err := json.Unmarshal(valueBytes, &capabilityUpdate); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid capabilityUpdate JSON: %v", err))
				}
				wh.handleBusinessCapabilitySubscriptionEvents(events.BaseBusinessAccountEvent{
					BusinessAccountId: entry.Id,
					Timestamp:         entryTime(entry.Time),
				}, capabilityUpdate)
				if err != nil {
					c.String(http.StatusInternalServerError, "Internal server error")
					return err
				}
			case WebhookFieldEnumSecurity:
				var securityChange SecurityValue
				valueBytes, err := json.Marshal(change.Value)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error marshaling messages value")
				}
				if err := json.Unmarshal(valueBytes, &securityChange); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid securityChange JSON: %v", err))
				}
				wh.handleSecuritySubscriptionEvents(securityChange)
				if err != nil {
					c.String(http.StatusInternalServerError, "Internal server error")
					return err
				}
			default:
				raw, err := json.Marshal(change.Value)
				if err != nil {
					return err
				}
				wh.EventManager.Publish(events.UnknownEventType, &events.UnknownWebhookEvent{Field: string(change.Field), BaseMessageEvent: events.BaseMessageEvent{BusinessAccountId: entry.Id}, Raw: raw})
			}
		}
	}

	c.String(200, "Message received")
	return nil
}

// entryTime returns the entry's unix time as a string, or "" when Meta omitted it.
func entryTime(t *json.Number) string {
	if t == nil {
		return ""
	}
	return t.String()
}

// ListenToEvents starts listening to events and handles incoming requests.
func (wh *WebhookManager) ListenToEvents() {
	fmt.Println("Listening to events")
	server := wh.createEchoHttpServer()
	server.GET(wh.path, wh.GetRequestHandler)
	server.POST(wh.path, wh.PostRequestHandler)

	// Start server in a goroutine
	go func() {
		if err := server.Start("127.0.0.1:8080"); err != nil {
			return
		}
	}()

	wh.EventManager.Publish(events.ReadyEventType, events.NewReadyEvent())
	// Wait for an interrupt signal (e.g., Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt) // Capture SIGINT (Ctrl+C)
	<-quit                            // Wait for the signal

	// Gracefully shut down the server (optional)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err) // Handle shutdown errors gracefully
	}
}

type HandleMessageSubscriptionEventPayload struct {
	Contacts          []Contact                  `json:"contacts"`
	Messages          []Message                  `json:"messages"`
	Statuses          []Status                   `json:"statuses"`
	PhoneNumber       events.BusinessPhoneNumber `json:"phone_number_id"`     // * this is the phone number to which this event has bee sent to
	BusinessAccountId string                     `json:"business_account_id"` // * business account id to which this event has been sent to
	SenderName        string                     `json:"sender_name"`
	UserActions       []UserAction               `json:"user_actions"`
}

func (wh *WebhookManager) handleMessagesSubscriptionEvents(payload HandleMessageSubscriptionEventPayload) error {
	// consider the field here too, because we will be supporting more events
	if len(payload.Statuses) > 0 {
		for _, status := range payload.Statuses {
			switch status.Status {
			case string(MessageStatusDelivered):
				{
					wh.EventManager.Publish(events.MessageDeliveredEventType, events.NewMessageDeliveredEvent(events.BaseSystemEvent{
						Timestamp: status.Timestamp,
					}, status.Id, status.RecipientId))
				}

			case string(MessageStatusRead):
				{
					wh.EventManager.Publish(events.MessageReadEventType, events.NewMessageReadEvent(events.BaseSystemEvent{
						Timestamp: status.Timestamp,
					}, status.Id, status.RecipientId))
				}
			case string(MessageStatusSent):
				{
					wh.EventManager.Publish(events.MessageSentEventType, events.NewMessageSentEvent(events.BaseSystemEvent{
						Timestamp: status.Timestamp,
					}, status.Id, status.RecipientId))
				}
			case string(MessageStatusFailed):
				{
					failedReason := ""
					errorCode := 0
					errorMessage := ""
					if len(status.Errors) > 0 {
						for _, err := range status.Errors {
							failedReason = err.Title
							errorCode = err.Code
							errorMessage = err.Message
							break
						}
					}

					wh.EventManager.Publish(events.MessageFailedEventType, events.NewMessageFailedEvent(events.BaseSystemEvent{
						Timestamp: status.Timestamp,
					}, status.Id, status.RecipientId, failedReason, errorCode, errorMessage))
				}
			case string(MessageStatusUnDelivered):
				{
					undeliveredReason := ""
					errorCode := 0
					errorMessage := ""
					if len(status.Errors) > 0 {
						for _, err := range status.Errors {
							undeliveredReason = err.Title
							errorCode = err.Code
							errorMessage = err.Message
							break
						}
					}

					wh.EventManager.Publish(events.MessageUndeliveredEventType, events.NewMessageUndeliveredEvent(events.BaseSystemEvent{
						Timestamp: status.Timestamp,
					}, status.Id, status.RecipientId, undeliveredReason, errorCode, errorMessage))
				}
			}

		}
	}

	for _, message := range payload.Messages {
		var repliedTo string
		if message.Context.Id != "" {
			repliedTo = message.Context.Id
		}

		senderName := payload.SenderName
		if len(payload.Contacts) > 0 {
			senderName = ""
			for _, contact := range payload.Contacts {
				if message.From != "" && contact.WaId == message.From {
					senderName = contact.Profile.Name
					break
				}
			}
		}
		baseMessageEvent := events.NewBaseMessageEvent(events.BaseMessageEventParams{
			BusinessAccountId: payload.BusinessAccountId,
			MessageId:         message.Id,
			PhoneNumber:       payload.PhoneNumber,
			Timestamp:         message.Timestamp,
			From:              message.From,
			SenderName:        senderName,
			IsForwarded:       message.Context.Forwarded,
			Context: events.MessageContext{
				RepliedToMessageId: repliedTo,
			},
			Requester: wh.Requester,
		})

		baseMessageEvent.RawMessage = message.RawMessage
		var rawFields map[string]json.RawMessage
		if json.Unmarshal(message.RawMessage, &rawFields) == nil {
			baseMessageEvent.Referral = rawFields["referral"]
		}

		switch message.Type {
		case NotificationMessageTypeText:
			{
				wh.EventManager.Publish(events.TextMessageEventType, events.NewTextMessageEvent(
					baseMessageEvent,
					message.Text.Body),
				)
			}
		case NotificationMessageTypeImage:
			{
				imageMessageComponent, err := components.NewImageMessage(components.ImageMessageConfigs{
					Id:      message.Image.Id,
					Caption: message.Image.Caption,
				})

				if err != nil {
					// ! TODO: emit error event here
					return err
				}

				wh.EventManager.Publish(events.ImageMessageEventType, events.NewImageMessageEvent(
					baseMessageEvent,
					*imageMessageComponent,
					message.Image.MIMEType, message.Image.SHA256, message.Image.Id),
				)
			}
		case NotificationMessageTypeAudio:
			{

				audioMessageComponent, err := components.NewAudioMessage(components.AudioMessageConfigs{
					Id: message.Audio.Id,
				})

				if err != nil {
					// ! TODO: emit error event here
					return err
				}

				wh.EventManager.Publish(events.AudioMessageEventType, events.NewAudioMessageEvent(
					baseMessageEvent,
					*audioMessageComponent,
					message.Audio.MIMEType, message.Audio.SHA256, message.Audio.Id),
				)

			}
		case NotificationMessageTypeVideo:
			{

				videoMessageComponent, err := components.NewVideoMessage(components.VideoMessageConfigs{
					Id:      message.Video.Id,
					Caption: message.Video.Caption,
				})

				if err != nil {
					// ! TODO: emit error event here
					return err
				}

				wh.EventManager.Publish(events.VideoMessageEventType, events.NewVideoMessageEvent(
					baseMessageEvent,
					*videoMessageComponent,
					message.Video.MIMEType, message.Video.SHA256, message.Video.Id),
				)

			}
		case NotificationMessageTypeDocument:
			// Inbound filenames and captions are optional; outbound constructor
			// validation must not reject a valid incoming document.
			document := components.DocumentMessage{Id: message.Document.Id, FileName: message.Document.Filename}
			if message.Document.Caption != "" {
				document.Caption = &message.Document.Caption
			}
			wh.EventManager.Publish(events.DocumentMessageEventType, events.NewDocumentMessageEvent(
				baseMessageEvent, document, message.Document.Id, message.Document.SHA256, message.Document.MIMEType))
		case NotificationMessageTypeLocation:
			{
				locationMessageComponent, err := components.NewLocationMessage(message.Location.Latitude, message.Location.Longitude)

				if err != nil {
					// ! TODO: emit error event here
					return err
				}

				wh.EventManager.Publish(events.LocationMessageEventType, events.NewLocationMessageEvent(
					baseMessageEvent,
					*locationMessageComponent),
				)
			}
		case NotificationMessageTypeContacts:
			{
				contactMessageComponent, _ := components.NewContactMessage(message.ContactCards)
				wh.EventManager.Publish(events.ContactMessageEventType, events.NewContactsMessageEvent(
					baseMessageEvent,
					*contactMessageComponent,
				))
			}
		case NotificationMessageTypeSticker:
			{

				stickerMessageComponent, err := components.NewStickerMessage(&components.StickerMessageConfigs{
					Id: message.Sticker.Id,
				})

				if err != nil {
					// ! TODO: emit error event here
					return err
				}

				wh.EventManager.Publish(events.StickerMessageEventType, events.NewStickerMessageEvent(
					baseMessageEvent,
					*stickerMessageComponent,
					message.Sticker.MIMEType, message.Sticker.SHA256, message.Sticker.Id),
				)

			}
		case NotificationMessageTypeButton:
			{
				wh.EventManager.Publish(events.QuickReplyMessageEventType, events.NewQuickReplyButtonInteractionEvent(
					baseMessageEvent,
					message.Button.Text,
					message.Button.Payload,
				))
			}
		case NotificationMessageTypeInteractive:
			{
				if message.Interactive.Type == "list_reply" {
					wh.EventManager.Publish(events.ListInteractionMessageEventType, events.NewListInteractionEvent(
						baseMessageEvent,
						message.Interactive.ListReply.Title,
						message.Interactive.ListReply.Id,
						message.Interactive.ListReply.Description,
					))
				} else {
					wh.EventManager.Publish(events.ReplyButtonInteractionEventType, events.NewReplyButtonInteractionEvent(
						baseMessageEvent,
						message.Interactive.ButtonReply.Title,
						message.Interactive.ButtonReply.Id,
					))
				}

			}
		case NotificationMessageTypeReaction:
			{
				reactionMessageComponent, err := components.NewReactionMessage(components.ReactionMessageParams{
					MessageId: message.Reaction.MessageId,
					Emoji:     message.Reaction.Emoji,
				})

				if err != nil {
					// ! TODO: emit error event here
					return err
				}

				wh.EventManager.Publish(events.ReactionMessageEventType, events.NewReactionMessageEvent(
					baseMessageEvent,
					*reactionMessageComponent,
				))
			}
		case NotificationMessageTypeOrder:
			{

				productItems := make([]components.ProductItem, len(message.Order.ProductItems))
				for i, item := range message.Order.ProductItems {
					productItems[i] = components.ProductItem{
						Currency:          item.Currency,
						ItemPrice:         item.ItemPrice,
						ProductRetailerID: item.ProductRetailerId,
						Quantity:          item.Quantity,
					}
				}

				wh.EventManager.Publish(events.OrderReceivedEventType, events.NewOrderEvent(
					baseMessageEvent,
					components.Order{
						CatalogID:    message.Order.CatalogId,
						ProductItems: productItems,
						Text:         message.Order.Text,
					},
				))
			}
		case NotificationMessageTypeSystem:
			{
				if message.System.Type == SystemNotificationTypeCustomerIdentityChanged {
					wh.EventManager.Publish(events.CustomerIdentityChangedEventType, events.CustomerIdentityChangedEvent{
						BaseSystemEvent: events.BaseSystemEvent{
							Timestamp: message.Timestamp,
						},
						Acknowledged:      message.Identity.Acknowledged,
						CreationTimestamp: message.Identity.CreatedTimestamp,
						Hash:              message.Identity.Hash,
					})
				} else if message.System.Type == SystemNotificationTypeCustomerPhoneNumberChange {
					wh.EventManager.Publish(events.CustomerNumberChangedEventType, events.CustomerNumberChangedEvent{
						BaseSystemEvent: events.BaseSystemEvent{
							Timestamp: message.Timestamp,
						},
						NewWaId:           message.System.WaId,
						OldWaId:           message.System.Customer,
						ChangeDescription: message.System.Body,
					})
				} else {
					wh.EventManager.Publish(events.UnknownEventType, &events.UnknownWebhookEvent{BaseMessageEvent: baseMessageEvent, Field: "messages", Raw: message.RawMessage})
				}
			}
		default:
			wh.EventManager.Publish(events.UnknownEventType, &events.UnknownWebhookEvent{BaseMessageEvent: baseMessageEvent, Field: "messages", Raw: message.RawMessage})
		}
	}

	// Handle user actions (e.g., marketing messages link clicks)
	for _, action := range payload.UserActions {
		switch action.ActionType {
		case UserActionTypeMarketingMessagesLinkClick:
			wh.EventManager.Publish(events.MarketingMessagesLinkClickEventType,
				events.NewMarketingMessagesLinkClickEvent(
					events.BaseBusinessAccountEvent{
						BusinessAccountId: payload.BusinessAccountId,
						Timestamp:         action.Timestamp,
					},
					payload.PhoneNumber,
					events.MarketingMessagesLinkClickData{
						ClickComponent: events.MarketingLinkClickComponent(action.MarketingMessagesLinkClickData.ClickComponent),
						ProductId:      action.MarketingMessagesLinkClickData.ProductId,
						ClickId:        action.MarketingMessagesLinkClickData.ClickId,
						TrackingToken:  action.MarketingMessagesLinkClickData.TrackingToken,
					},
				))
		}
	}

	return nil
}

func (wh *WebhookManager) handleAccountAlertsSubscriptionEvents(baseEvent events.BaseBusinessAccountEvent, value AccountAlertsValue) error {
	wh.EventManager.Publish(events.AccountAlertsEventType, events.NewAccountAlertEvent(
		&baseEvent,
		value.EntityType,
		value.EntityId,
		events.AccountAlertSeverityEnum(value.AlertSeverity),
		events.AccountAlertStatusEnum(value.AlertStatus),
		value.AlertType,
		value.AlertDescription,
	))
	return nil
}

func (wh *WebhookManager) handleSecuritySubscriptionEvents(value SecurityValue) {
	wh.EventManager.Publish(events.SecurityEventType, events.SecurityEvent{})

}

func (wh *WebhookManager) handleAccountUpdateSubscriptionEvents(baseEvent events.BaseBusinessAccountEvent, value AccountUpdateValue) {

	wh.EventManager.Publish(events.AccountUpdateEventType, events.NewAccountUpdateEvent(
		&baseEvent,
		events.AccountUpdateEventEnum(value.Event),
		value.PhoneNumber,
	))

}

func (wh *WebhookManager) handleAccountReviewSubscriptionEvents(baseEvent events.BaseBusinessAccountEvent, value AccountReviewUpdateValue) error {
	wh.EventManager.Publish(events.AccountReviewUpdateEventType, events.NewAccountReviewUpdateEvent(
		&baseEvent,
		events.AccountReviewUpdateEventEnum(value.Decision),
	))
	return nil

}

func (wh *WebhookManager) handleBusinessCapabilitySubscriptionEvents(baseEvent events.BaseBusinessAccountEvent, value BusinessCapabilityUpdateValue) error {
	wh.EventManager.Publish(events.BusinessCapabilityUpdateEventType, events.NewBusinessCapabilityUpdateEvent(
		&baseEvent,
		int64(value.MaxDailyConversationPerPhone),
		int64(value.MaxPhoneNumbersPerBusiness),
	))
	return nil

}

func (wh *WebhookManager) handleMessageTemplateQualitySubscriptionEvents(baseEvent events.BaseBusinessAccountEvent, value TemplateQualityUpdateValue) error {
	wh.EventManager.Publish(events.MessageTemplateQualityUpdateEventType, events.NewMessageTemplateQualityUpdateEvent(
		&baseEvent,
		events.MessageTemplateQualityUpdateQualityScoreEnum(value.PreviousQualityScore),
		events.MessageTemplateQualityUpdateQualityScoreEnum(value.NewQualityScore),
		value.MessageTemplateId,
		value.MessageTemplateName,
		value.MessageTemplateLanguage,
	))

	return nil

}

func (wh *WebhookManager) handleMessageTemplateStatusSubscriptionEvents(baseEvent events.BaseBusinessAccountEvent, value TemplateStatusUpdateValue) error {
	wh.EventManager.Publish(events.MessageTemplateStatusUpdateEventType, events.NewMessageTemplateStatusUpdateEvent(
		&baseEvent,
		events.MessageTemplateStatusUpdateEventEnum(value.Event),
		value.MessageTemplateId,
		value.MessageTemplateName,
		value.MessageTemplateLanguage,
		events.MessageTemplateStatusUpdateReason(value.Reason),
	))
	return nil

}

func (wh *WebhookManager) handlePhoneNumberNameSubscriptionEvents(baseEvent events.BaseBusinessAccountEvent, value PhoneNumberNameUpdateValue) error {
	wh.EventManager.Publish(events.PhoneNumberNameUpdateEventType, events.NewPhoneNumberNameUpdateEvent(
		&baseEvent,
		value.DisplayPhoneNumber,
		value.RequestedVerifiedName,
		value.Decision,
		&value.RejectionReason,
	))
	return nil
}

func (wh *WebhookManager) handlePhoneNumberQualitySubscriptionEvents(baseEvent events.BaseBusinessAccountEvent, value PhoneNumberQualityUpdateValue) error {
	wh.EventManager.Publish(events.PhoneNumberQualityUpdateEventType, events.NewPhoneNumberQualityUpdateEvent(
		&baseEvent,
		value.DisplayPhoneNumber,
		events.PhoneNumberUpdateEventEnum(value.Event),
		events.PhoneNumberQualityUpdateCurrentLimitEnum(value.CurrentLimit),
	))
	return nil
}

func (wh *WebhookManager) handleTemplateCategoryUpdateSubscriptionEvents(baseEvent events.BaseBusinessAccountEvent, value TemplateCategoryUpdateValue) error {
	wh.EventManager.Publish(events.TemplateCategoryUpdateEventType, events.NewMessageTemplateCategoryUpdateEvent(
		&baseEvent,
		value.MessageTemplateId,
		value.MessageTemplateName,
		value.MessageTemplateLanguage,
		events.MessageTemplateCategoryEnum(value.PreviousCategory),
		events.MessageTemplateCategoryEnum(value.NewCategory),
	))
	return nil
}
