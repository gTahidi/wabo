package components

import (
	"encoding/json"
	"fmt"

	"github.com/gTahidi/wabo/internal"
)

// FlowMessageMode represents the mode for sending a flow
type FlowMessageMode string

const (
	// FlowMessageModeDraft sends the current draft version of the flow
	FlowMessageModeDraft FlowMessageMode = "draft"
	// FlowMessageModePublished sends the last published version of the flow
	FlowMessageModePublished FlowMessageMode = "published"
)

// FlowAction represents the action type for a flow
type FlowAction string

const (
	// FlowActionNavigate triggers navigation to another screen with static payload
	FlowActionNavigate FlowAction = "navigate"
	// FlowActionDataExchange sends data to WhatsApp Flows Data Endpoint
	FlowActionDataExchange FlowAction = "data_exchange"
)

// FlowActionPayload contains the payload for navigate action
type FlowActionPayload struct {
	// Screen is the unique ID of the Screen to navigate to
	// Default is "FIRST_ENTRY_SCREEN"
	Screen string `json:"screen,omitempty"`
	// Data is the input data for the screen
	Data map[string]interface{} `json:"data,omitempty"`
}

// FlowMessageActionParams contains the parameters for a flow action
// See: https://developers.facebook.com/docs/whatsapp/flows/gettingstarted/sendingaflow
type FlowMessageActionParams struct {
	// FlowMessageVersion is required, typically "3"
	FlowMessageVersion string `json:"flow_message_version" validate:"required"`
	// FlowID is the unique identifier for the flow (either FlowID or FlowName required)
	FlowID string `json:"flow_id,omitempty"`
	// FlowName is the name of the flow (either FlowID or FlowName required)
	FlowName string `json:"flow_name,omitempty"`
	// FlowCTA is the call to action button text (required)
	FlowCTA string `json:"flow_cta" validate:"required"`
	// FlowToken is an optional token to authenticate the flow, default is "unused"
	FlowToken string `json:"flow_token,omitempty"`
	// FlowAction is either "navigate" or "data_exchange", default is "navigate"
	FlowAction FlowAction `json:"flow_action,omitempty"`
	// FlowActionPayload contains screen and data for navigate action
	FlowActionPayload *FlowActionPayload `json:"flow_action_payload,omitempty"`
	// Mode is "draft" or "published", default is "published"
	Mode FlowMessageMode `json:"mode,omitempty"`
}

// FlowMessageAction represents the action for a flow message
type FlowMessageAction struct {
	// Name must be "flow"
	Name string `json:"name" validate:"required"`
	// Parameters contains the flow action parameters
	Parameters FlowMessageActionParams `json:"parameters" validate:"required"`
}

// FlowMessageBody represents the body of a flow message
type FlowMessageBody struct {
	Text string `json:"text" validate:"required"`
}

// FlowMessageFooter represents the footer of a flow message
type FlowMessageFooter struct {
	Text string `json:"text"`
}

// FlowMessageHeaderType defines the type of header
type FlowMessageHeaderType string

const (
	FlowMessageHeaderTypeText FlowMessageHeaderType = "text"
)

// FlowMessageHeader represents the header of a flow message
type FlowMessageHeader struct {
	Type FlowMessageHeaderType `json:"type" validate:"required"`
	Text string                `json:"text,omitempty"`
}

// FlowMessage represents an interactive flow message
// See: https://developers.facebook.com/docs/whatsapp/flows/gettingstarted/sendingaflow
type FlowMessage struct {
	Type   InteractiveMessageType `json:"type" validate:"required"`
	Header *FlowMessageHeader     `json:"header,omitempty"`
	Body   FlowMessageBody        `json:"body" validate:"required"`
	Footer *FlowMessageFooter     `json:"footer,omitempty"`
	Action FlowMessageAction      `json:"action" validate:"required"`
}

// FlowMessageParams contains parameters for creating a new flow message
type FlowMessageParams struct {
	// BodyText is the message body text (required)
	BodyText string `validate:"required"`
	// FlowID is the unique ID of the flow (either FlowID or FlowName required)
	FlowID string
	// FlowName is the name of the flow (either FlowID or FlowName required)
	FlowName string
	// FlowCTA is the call to action button text (required)
	FlowCTA string `validate:"required"`
	// FlowMessageVersion is the version of flow message protocol, default is "3"
	FlowMessageVersion string
}

// FlowMessageApiPayload represents the API payload for a flow message
type FlowMessageApiPayload struct {
	BaseMessagePayload
	Interactive FlowMessage `json:"interactive" validate:"required"`
}

// NewFlowMessage creates a new flow message for user-initiated conversations
func NewFlowMessage(params FlowMessageParams) (*FlowMessage, error) {
	if err := internal.GetValidator().Struct(params); err != nil {
		return nil, fmt.Errorf("error validating params: %v", err)
	}

	if params.FlowID == "" && params.FlowName == "" {
		return nil, fmt.Errorf("either FlowID or FlowName is required")
	}

	version := params.FlowMessageVersion
	if version == "" {
		version = "3"
	}

	msg := &FlowMessage{
		Type: InteractiveMessageTypeFlow,
		Body: FlowMessageBody{Text: params.BodyText},
		Action: FlowMessageAction{
			Name: "flow",
			Parameters: FlowMessageActionParams{
				FlowMessageVersion: version,
				FlowCTA:            params.FlowCTA,
			},
		},
	}

	if params.FlowID != "" {
		msg.Action.Parameters.FlowID = params.FlowID
	} else {
		msg.Action.Parameters.FlowName = params.FlowName
	}

	return msg, nil
}

// SetHeader sets the text header for the flow message
func (m *FlowMessage) SetHeader(text string) {
	m.Header = &FlowMessageHeader{
		Type: FlowMessageHeaderTypeText,
		Text: text,
	}
}

// SetFooter sets the footer text for the flow message
func (m *FlowMessage) SetFooter(text string) {
	m.Footer = &FlowMessageFooter{Text: text}
}

// SetFlowToken sets the flow token for authentication
func (m *FlowMessage) SetFlowToken(token string) {
	m.Action.Parameters.FlowToken = token
}

// SetMode sets the flow mode (draft or published)
func (m *FlowMessage) SetMode(mode FlowMessageMode) {
	m.Action.Parameters.Mode = mode
}

// SetFlowAction sets the flow action and optional payload
func (m *FlowMessage) SetFlowAction(action FlowAction, payload *FlowActionPayload) {
	m.Action.Parameters.FlowAction = action
	m.Action.Parameters.FlowActionPayload = payload
}

// ToJson converts the flow message to JSON for the WhatsApp Cloud API
func (m *FlowMessage) ToJson(configs ApiCompatibleJsonConverterConfigs) ([]byte, error) {
	if err := internal.GetValidator().Struct(configs); err != nil {
		return nil, fmt.Errorf("error validating configs: %v", err)
	}

	jsonData := FlowMessageApiPayload{
		BaseMessagePayload: NewBaseMessagePayload(configs.SendToPhoneNumber, MessageTypeInteractive),
		Interactive:        *m,
	}

	if configs.ReplyToMessageId != "" {
		jsonData.Context = &Context{MessageId: configs.ReplyToMessageId}
	}

	jsonToReturn, err := json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("error marshalling json: %v", err)
	}

	return jsonToReturn, nil
}
