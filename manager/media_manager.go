package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gTahidi/wapi.go/internal/request_client"
	"github.com/gTahidi/wapi.go/pkg/graph"
)

// MediaManager is responsible for managing media related operations.
type MediaManager struct {
	requester request_client.RequestClient
}

// NewMediaManager creates a new instance of MediaManager.
func NewMediaManager(requester request_client.RequestClient) *MediaManager {
	return &MediaManager{
		requester: requester,
	}
}

type MediaMetadata struct {
	MessagingProduct string `json:"messaging_product"`
	Url              string `json:"url"`
	MimeType         string `json:"mime_type"`
	Sha256           string `json:"sha256"`
	FileSize         int    `json:"file_size"`
	ID               string `json:"id"`
}

func (mm *MediaManager) GetMediaUrlById(id string) (string, error) {
	// Build GET request to: e.g. "<MEDIA_ID>" (the request client automatically prefixes the base URL and version)
	apiRequest := mm.requester.NewApiRequest(id, http.MethodGet)

	// Execute the request and get the raw JSON response
	rawResponse, err := apiRequest.Execute()
	if err != nil {
		return "", err
	}

	// Parse into a struct
	var res MediaMetadata
	if err := json.Unmarshal([]byte(rawResponse), &res); err != nil {
		return "", fmt.Errorf("failed to parse media metadata: %w", err)
	}

	if res.Url == "" {
		return "", fmt.Errorf("no media url found in response: %s", rawResponse)
	}

	return res.Url, nil
}

type DeleteSuccessResponse struct {
	Success bool `json:"success"`
}

func (mm *MediaManager) DeleteMedia(id string) (string, error) {
	// The path becomes "media/<MEDIA_ID>"
	apiRequest := mm.requester.NewApiRequest(strings.Join([]string{"media", id}, "/"), http.MethodDelete)

	rawResponse, err := apiRequest.Execute()
	if err != nil {
		return "", err
	}

	// Parse the JSON
	var res DeleteSuccessResponse
	if err := json.Unmarshal([]byte(rawResponse), &res); err != nil {
		return "", fmt.Errorf("failed to parse delete response: %w", err)
	}

	if !res.Success {
		return "", fmt.Errorf("media deletion failed or returned success=false: %s", rawResponse)
	}

	return "media deleted successfully", nil
}

// UploadMedia uploads a media file to WhatsApp's Cloud API.
func (mm *MediaManager) UploadMedia(phoneNumberId string, file io.Reader, filename, mimeType string) (string, error) {
	// 1. Build the multipart form in memory
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("messaging_product", "whatsapp"); err != nil {
		return "", fmt.Errorf("failed to write field: %w", err)
	}

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filepath.Base(filename)))
	partHeader.Set("Content-Type", mimeType)

	filePart, err := writer.CreatePart(partHeader)
	if err != nil {
		return "", fmt.Errorf("failed to create multipart part: %w", err)
	}
	if _, err := io.Copy(filePart, file); err != nil {
		return "", fmt.Errorf("failed to copy file into part: %w", err)
	}

	// Close the writer to finalize the multipart data
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	apiPath := strings.Join([]string{phoneNumberId, "media"}, "/")

	contentType := writer.FormDataContentType()

	responseBody, err := mm.requester.RequestMultipart(http.MethodPost, apiPath, body, contentType)
	if err != nil {
		return "", fmt.Errorf("error uploading media: %w", err)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(responseBody), &result); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}
	if result.ID == "" {
		// Possibly an error or partial success
		return "", fmt.Errorf("no media id in response: %s", responseBody)
	}

	return result.ID, nil
}

// ResumableUploadSession represents the response from creating an upload session
type ResumableUploadSession struct {
	ID string `json:"id"` // The upload session ID (h:xxx format)
}

// ResumableUploadResult represents the response from completing an upload
type ResumableUploadResult struct {
	Handle string `json:"h"` // The media handle to use in templates (4::xxx format)
}

// CreateResumableUploadSession creates a new resumable upload session for template media.
// This is used to upload media files that will be used in message template headers.
// appID is your Meta App ID, fileLength is the size in bytes, fileType is the MIME type (e.g., "image/png").
// Returns the upload session ID.
func (mm *MediaManager) CreateResumableUploadSession(appID string, fileLength int64, fileType string) (string, error) {
	// POST to /{app-id}/uploads with file_length, file_type, file_name
	path := fmt.Sprintf("%s/uploads", appID)

	body := map[string]interface{}{
		"file_length": fileLength,
		"file_type":   fileType,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	apiRequest := mm.requester.NewApiRequest(path, http.MethodPost)
	apiRequest.SetBody(string(bodyJSON))

	rawResponse, err := apiRequest.Execute()
	if err != nil {
		return "", fmt.Errorf("failed to create upload session: %w", err)
	}

	var result ResumableUploadSession
	if err := json.Unmarshal([]byte(rawResponse), &result); err != nil {
		return "", fmt.Errorf("failed to parse upload session response: %w", err)
	}

	if result.ID == "" {
		return "", fmt.Errorf("no upload session ID in response: %s", rawResponse)
	}

	return result.ID, nil
}

// UploadResumableMedia uploads file data to an existing upload session.
// sessionID is the upload session ID from CreateResumableUploadSession.
// fileData is the raw file bytes, fileOffset is the starting byte offset (usually 0).
// Returns the media handle to use in template header_handle.
func (mm *MediaManager) UploadResumableMedia(sessionID string, fileData []byte, fileOffset int64) (string, error) {
	// POST to /{upload-session-id} with file data in body
	// Headers: Authorization, file_offset

	requestPath := fmt.Sprintf("%s://%s/%s/%s",
		request_client.REQUEST_PROTOCOL,
		request_client.BASE_URL,
		request_client.API_VERSION,
		sessionID,
	)

	httpRequest, err := http.NewRequest(http.MethodPost, requestPath, bytes.NewReader(fileData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpRequest.Header.Set("Authorization", fmt.Sprintf("OAuth %s", mm.requester.ApiAccessToken()))
	httpRequest.Header.Set("file_offset", strconv.FormatInt(fileOffset, 10))

	response, err := mm.requester.HTTPClient().Do(httpRequest)
	if err != nil {
		return "", fmt.Errorf("failed to execute upload request: %w", err)
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", graph.Parse(response.StatusCode, respBody)
	}

	var result ResumableUploadResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse upload response: %w", err)
	}

	if result.Handle == "" {
		return "", fmt.Errorf("no media handle in response: %s", string(respBody))
	}

	return result.Handle, nil
}

// UploadMediaForTemplate uploads a file and returns a handle suitable for use in template header_handle.
// This is a convenience method that combines CreateResumableUploadSession and UploadResumableMedia.
// appID is your Meta App ID, fileData is the raw file bytes, fileType is the MIME type.
func (mm *MediaManager) UploadMediaForTemplate(appID string, fileData []byte, fileType string) (string, error) {
	// Step 1: Create upload session
	sessionID, err := mm.CreateResumableUploadSession(appID, int64(len(fileData)), fileType)
	if err != nil {
		return "", fmt.Errorf("failed to create upload session: %w", err)
	}

	// Step 2: Upload the file data
	handle, err := mm.UploadResumableMedia(sessionID, fileData, 0)
	if err != nil {
		return "", fmt.Errorf("failed to upload media: %w", err)
	}

	return handle, nil
}
