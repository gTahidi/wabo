package request_client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gTahidi/wabo/pkg/graph"
)

const (
	API_VERSION      = "v24.0"
	BASE_URL         = "graph.facebook.com"
	REQUEST_PROTOCOL = "https"

	// DefaultTimeout bounds a Graph API call when the caller's context has no
	// earlier deadline.
	DefaultTimeout = 30 * time.Second

	// maxResponseBytes caps how much of a response body is read.
	maxResponseBytes = 16 << 20
)

type WhatsappApiType string

const (
	WhatsappApiTypeMessaging WhatsappApiType = "messaging"
	WhatsappApiTypeBusiness  WhatsappApiType = "business"
)

// RequestClient represents a client for making requests to a cloud API.
type RequestClient struct {
	apiVersion     string
	baseUrl        string
	apiAccessToken string
	httpClient     *http.Client
}

func (client *RequestClient) BaseUrl() string {
	return client.baseUrl
}

func (client *RequestClient) ApiVersion() string {
	return client.apiVersion
}

func (client *RequestClient) ApiAccessToken() string {
	return client.apiAccessToken
}

// HTTPClient returns the client used for Graph API calls.
func (client *RequestClient) HTTPClient() *http.Client {
	if client.httpClient == nil {
		return defaultHTTPClient
	}
	return client.httpClient
}

var defaultHTTPClient = &http.Client{Timeout: DefaultTimeout}

// NewRequestClient creates a new instance of RequestClient.
func NewRequestClient(apiAccessToken string) *RequestClient {
	return NewRequestClientWithHTTPClient(apiAccessToken, nil)
}

// NewRequestClientWithHTTPClient creates a RequestClient that sends through
// httpClient (nil uses a shared client with DefaultTimeout).
func NewRequestClientWithHTTPClient(apiAccessToken string, httpClient *http.Client) *RequestClient {
	return NewRequestClientWithVersion(apiAccessToken, httpClient, "")
}

// NewRequestClientWithVersion configures the Graph version (for example v26.0).
// An empty version retains the existing default.
func NewRequestClientWithVersion(apiAccessToken string, httpClient *http.Client, apiVersion string) *RequestClient {
	if apiVersion == "" {
		apiVersion = API_VERSION
	}
	return &RequestClient{
		apiVersion:     apiVersion,
		baseUrl:        BASE_URL,
		apiAccessToken: apiAccessToken,
		httpClient:     httpClient,
	}
}

// RequestCloudApiParams represents the parameters for making a request to the cloud API.
type RequestCloudApiParams struct {
	Body       string
	Path       string
	Method     string
	QueryParam map[string]string
}

func (requestClientInstance *RequestClient) url(path string, query map[string]string) string {
	requestPath := strings.Join(
		[]string{REQUEST_PROTOCOL, "://", requestClientInstance.baseUrl, "/", requestClientInstance.apiVersion, "/", path}, "")
	if len(query) > 0 {
		values := url.Values{}
		for key, value := range query {
			values.Set(key, value)
		}
		requestPath += "?" + values.Encode()
	}
	return requestPath
}

func (requestClientInstance *RequestClient) request(ctx context.Context, params RequestCloudApiParams) (string, error) {
	var body io.Reader
	if params.Body != "" {
		body = strings.NewReader(params.Body)
	}
	return requestClientInstance.do(ctx, params.Method, requestClientInstance.url(params.Path, params.QueryParam), body, "application/json")
}

// do sends one request and returns the response body. A non-2xx response
// returns a *graph.Error.
func (requestClientInstance *RequestClient) do(ctx context.Context, method, requestURL string, body io.Reader, contentType string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	httpRequest, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	if contentType != "" {
		httpRequest.Header.Set("Content-Type", contentType)
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", requestClientInstance.apiAccessToken))
	response, err := requestClientInstance.HTTPClient().Do(httpRequest)
	if err != nil {
		return "", fmt.Errorf("graph api request failed: %w", err)
	}
	defer response.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", graph.Parse(response.StatusCode, respBody)
	}

	return string(respBody), nil
}

func (client *RequestClient) NewApiRequest(path, method string) *ApiRequest {
	return &ApiRequest{
		Path:        path,
		Fields:      []*ApiRequestQueryParamField{}, // Changed to slice of pointers
		Requester:   client,
		Method:      method,
		QueryParams: map[string]string{},
	}
}

type ApiRequestQueryParamField struct {
	Name    string
	Filters map[string]string
}

func (field *ApiRequestQueryParamField) AddFilter(key, value string) {
	field.Filters[key] = value
}

type ApiRequest struct {
	Path        string
	Method      string
	Body        string
	Fields      []*ApiRequestQueryParamField // Changed to slice of pointers
	QueryParams map[string]string
	Requester   *RequestClient
}

func (request *ApiRequest) AddField(field ApiRequestQueryParamField) *ApiRequestQueryParamField {
	// * NOTE:  when we say we need to add a field to the request, it means we need to add a query param to the request with key "fields"
	// * note that if there need to be multiple fields in a single request then the list of fields should be command separated
	// * for example: fields=field1,field2,field3
	// * also note that if there filters in a field then they should be called like a function in the param string, for ex: fields=field1.filter1(value1).filter2(value2),field2.filter1(value1)

	// Store pointer to the field so subsequent modifications (AddFilter) affect the stored field
	ptr := &field
	request.Fields = append(request.Fields, ptr)
	return ptr
}

// AddQueryParam adds a query parameter to the request.
func (request *ApiRequest) AddQueryParam(key, value string) {
	request.QueryParams[key] = value
}

// SetMethod sets the method for the request.
func (request *ApiRequest) SetMethod(method string) {
	request.Method = method
}

// SetBody sets the body for the request.
func (request *ApiRequest) SetBody(body string) {
	request.Body = body
}

// Execute executes the request and returns the response.
func (request *ApiRequest) Execute() (string, error) {
	return request.ExecuteContext(context.Background())
}

// ExecuteContext executes the request, bounded by ctx, and returns the
// response body. A non-2xx response returns a *graph.Error.
func (request *ApiRequest) ExecuteContext(ctx context.Context) (string, error) {
	// check if there are any fields in the request
	var queryParam = map[string]string{}
	if len(request.Fields) > 0 {
		fieldsString := ""
		for _, field := range request.Fields {
			newFieldString := ""
			if fieldsString != "" {
				newFieldString = ","
			}
			filterString := ""
			for key, value := range field.Filters {
				filterString += strings.Join([]string{".", key, "(", value, ")"}, "")
			}
			newFieldString += strings.Join([]string{field.Name, filterString}, "")
			fieldsString += newFieldString
		}

		queryParam["fields"] = fieldsString
	}

	if len(request.QueryParams) > 0 {
		for key, value := range request.QueryParams {
			queryParam[key] = value
		}
	}

	return request.Requester.request(ctx, RequestCloudApiParams{
		Path:       request.Path,
		Body:       request.Body,
		Method:     request.Method,
		QueryParam: queryParam,
	})
}

// RequestMultipart allows sending an arbitrary body with a custom Content-Type.
// This is needed for file uploads (multipart/form-data).
func (rc *RequestClient) RequestMultipart(
	method string,
	path string,
	body io.Reader,
	contentType string,
) (string, error) {
	return rc.RequestMultipartContext(context.Background(), method, path, body, contentType)
}

// RequestMultipartContext is RequestMultipart bounded by ctx.
func (rc *RequestClient) RequestMultipartContext(
	ctx context.Context,
	method string,
	path string,
	body io.Reader,
	contentType string,
) (string, error) {
	return rc.do(ctx, method, rc.url(path, nil), body, contentType)
}
