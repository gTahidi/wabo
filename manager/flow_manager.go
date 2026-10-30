package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gTahidi/wabo/internal"
	"github.com/gTahidi/wabo/internal/request_client"
)

// FlowCategory represents the category of a WhatsApp Flow
type FlowCategory string

const (
	FlowCategorySignUp             FlowCategory = "SIGN_UP"
	FlowCategorySignIn             FlowCategory = "SIGN_IN"
	FlowCategoryAppointmentBooking FlowCategory = "APPOINTMENT_BOOKING"
	FlowCategoryLeadGeneration     FlowCategory = "LEAD_GENERATION"
	FlowCategoryContactUs          FlowCategory = "CONTACT_US"
	FlowCategoryCustomerSupport    FlowCategory = "CUSTOMER_SUPPORT"
	FlowCategorySurvey             FlowCategory = "SURVEY"
	FlowCategoryOther              FlowCategory = "OTHER"
)

// FlowStatus represents the status of a WhatsApp Flow
type FlowStatus string

const (
	FlowStatusDraft      FlowStatus = "DRAFT"
	FlowStatusPublished  FlowStatus = "PUBLISHED"
	FlowStatusDeprecated FlowStatus = "DEPRECATED"
	FlowStatusBlocked    FlowStatus = "BLOCKED"
	FlowStatusThrottled  FlowStatus = "THROTTLED"
)

type FlowValidationError struct {
	Error       string `json:"error"`
	ErrorType   string `json:"error_type"`
	Message     string `json:"message"`
	LineStart   int    `json:"line_start,omitempty"`
	LineEnd     int    `json:"line_end,omitempty"`
	ColumnStart int    `json:"column_start,omitempty"`
	ColumnEnd   int    `json:"column_end,omitempty"`
}

type FlowPreview struct {
	PreviewURL string `json:"preview_url"`
	ExpiresAt  string `json:"expires_at"`
}

type FlowHealthStatus struct {
	CanSend  bool                     `json:"can_send"`
	Entities []FlowHealthStatusEntity `json:"entities,omitempty"`
}
type FlowHealthStatusEntity struct {
	EntityType     string   `json:"entity_type"`
	ID             string   `json:"id"`
	CanSend        bool     `json:"can_send"`
	Errors         []string `json:"errors,omitempty"`
	AdditionalInfo string   `json:"additional_info,omitempty"`
}

type FlowNode struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	Status           FlowStatus            `json:"status"`
	Categories       []FlowCategory        `json:"categories"`
	ValidationErrors []FlowValidationError `json:"validation_errors,omitempty"`
	JSONVersion      string                `json:"json_version,omitempty"`
	DataAPIVersion   string                `json:"data_api_version,omitempty"`
	EndpointURI      string                `json:"endpoint_uri,omitempty"`
	Preview          *FlowPreview          `json:"preview,omitempty"`
	HealthStatus     *FlowHealthStatus     `json:"health_status,omitempty"`
}

type FlowsListResponse struct {
	Data   []FlowNode                                 `json:"data"`
	Paging internal.WhatsAppBusinessApiPaginationMeta `json:"paging,omitempty"`
}
type FlowManager struct {
	businessAccountId string
	apiAccessToken    string
	requester         *request_client.RequestClient
}
type FlowManagerConfig struct {
	BusinessAccountId string
	ApiAccessToken    string
	Requester         *request_client.RequestClient
}

func NewFlowManager(config *FlowManagerConfig) *FlowManager {
	return &FlowManager{
		businessAccountId: config.BusinessAccountId,
		apiAccessToken:    config.ApiAccessToken,
		requester:         config.Requester,
	}
}

type CreateFlowRequest struct {
	Name        string         `json:"name" validate:"required"`
	Categories  []FlowCategory `json:"categories" validate:"required,min=1"`
	FlowJSON    string         `json:"flow_json,omitempty"`
	Publish     bool           `json:"publish,omitempty"`
	CloneFlowID string         `json:"clone_flow_id,omitempty"`
	EndpointURI string         `json:"endpoint_uri,omitempty"`
}
type CreateFlowResponse struct {
	ID               string                `json:"id"`
	Success          bool                  `json:"success"`
	ValidationErrors []FlowValidationError `json:"validation_errors,omitempty"`
}

func (m *FlowManager) Create(req CreateFlowRequest) (*CreateFlowResponse, error) {
	apiRequest := m.requester.NewApiRequest(
		strings.Join([]string{m.businessAccountId, "flows"}, "/"),
		http.MethodPost,
	)

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiRequest.SetBody(string(jsonBody))
	response, err := apiRequest.Execute()
	if err != nil {
		return nil, err
	}

	var result CreateFlowResponse
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

func (m *FlowManager) FetchAll() (*FlowsListResponse, error) {
	apiRequest := m.requester.NewApiRequest(
		strings.Join([]string{m.businessAccountId, "flows"}, "/"),
		http.MethodGet,
	)

	response, err := apiRequest.Execute()
	if err != nil {
		return nil, err
	}

	var result FlowsListResponse
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

func (m *FlowManager) Fetch(flowID string) (*FlowNode, error) {
	fields := "id,name,status,categories,validation_errors,json_version,data_api_version,endpoint_uri,preview,health_status"
	apiRequest := m.requester.NewApiRequest(flowID, http.MethodGet)
	apiRequest.AddQueryParam("fields", fields)

	response, err := apiRequest.Execute()
	if err != nil {
		return nil, err
	}

	var result FlowNode
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

type UpdateFlowRequest struct {
	Name        string         `json:"name,omitempty"`
	Categories  []FlowCategory `json:"categories,omitempty"`
	EndpointURI string         `json:"endpoint_uri,omitempty"`
}

func (m *FlowManager) Update(flowID string, req UpdateFlowRequest) error {
	apiRequest := m.requester.NewApiRequest(flowID, http.MethodPost)

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	apiRequest.SetBody(string(jsonBody))
	_, err = apiRequest.Execute()
	return err
}

// UploadFlowJSONResponse represents the response from uploading flow JSON
type UploadFlowJSONResponse struct {
	Success          bool                  `json:"success"`
	ValidationErrors []FlowValidationError `json:"validation_errors,omitempty"`
}

// UploadFlowJSON uploads or updates the Flow JSON for an existing flow.
// Meta API requires multipart/form-data for the /{FLOW_ID}/assets endpoint.
func (m *FlowManager) UploadFlowJSON(flowID string, flowJSON string) (*UploadFlowJSONResponse, error) {
	// Build multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add 'name' field
	if err := writer.WriteField("name", "flow.json"); err != nil {
		return nil, fmt.Errorf("failed to write 'name' field: %w", err)
	}

	// Add 'asset_type' field
	if err := writer.WriteField("asset_type", "FLOW_JSON"); err != nil {
		return nil, fmt.Errorf("failed to write 'asset_type' field: %w", err)
	}

	// Add 'file' field with JSON content
	fileWriter, err := writer.CreateFormFile("file", "flow.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := fileWriter.Write([]byte(flowJSON)); err != nil {
		return nil, fmt.Errorf("failed to write flow JSON content: %w", err)
	}

	// Close the writer to finalize the multipart message
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Make the request using RequestMultipart
	path := strings.Join([]string{flowID, "assets"}, "/")
	response, err := m.requester.RequestMultipart(
		http.MethodPost,
		path,
		&buf,
		writer.FormDataContentType(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload flow JSON: %w", err)
	}

	var result UploadFlowJSONResponse
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

func (m *FlowManager) Publish(flowID string) error {
	apiRequest := m.requester.NewApiRequest(
		strings.Join([]string{flowID, "publish"}, "/"),
		http.MethodPost,
	)

	_, err := apiRequest.Execute()
	return err
}

func (m *FlowManager) Deprecate(flowID string) error {
	apiRequest := m.requester.NewApiRequest(
		strings.Join([]string{flowID, "deprecate"}, "/"),
		http.MethodPost,
	)

	_, err := apiRequest.Execute()
	return err
}

func (m *FlowManager) Delete(flowID string) error {
	apiRequest := m.requester.NewApiRequest(flowID, http.MethodDelete)

	_, err := apiRequest.Execute()
	return err
}

func (m *FlowManager) GetFlowJSON(flowID string) (string, error) {
	apiRequest := m.requester.NewApiRequest(
		strings.Join([]string{flowID, "assets"}, "/"),
		http.MethodGet,
	)

	response, err := apiRequest.Execute()
	if err != nil {
		return "", err
	}

	return response, nil
}
