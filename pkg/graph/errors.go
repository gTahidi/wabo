// Package graph holds the typed error returned when the Graph API answers
// with a non-2xx status, so callers can decide whether to retry, refresh the
// access token, or give up.
//
// Meta recommends handling errors by code and error_data.details;
// error_subcode is deprecated from v16.0. See
// https://developers.facebook.com/docs/whatsapp/cloud-api/support/error-codes
package graph

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Error is a Graph API error response.
type Error struct {
	// HTTPStatus is the HTTP status code of the response.
	HTTPStatus int    `json:"-"`
	Message    string `json:"message"`
	Type       string `json:"type"`
	Code       int    `json:"code"`
	ErrorData  struct {
		MessagingProduct string `json:"messaging_product"`
		Details          string `json:"details"`
	} `json:"error_data"`
	// ErrorSubcode is deprecated by Meta and not returned from v16.0.
	ErrorSubcode int    `json:"error_subcode"`
	ErrorUserMsg string `json:"error_user_msg"`
	FbtraceID    string `json:"fbtrace_id"`
}

func (e *Error) Error() string {
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.HTTPStatus)
	}
	s := fmt.Sprintf("graph api %d: %s (code %d", e.HTTPStatus, msg, e.Code)
	if e.ErrorData.Details != "" {
		s += ", details: " + e.ErrorData.Details
	}
	if e.FbtraceID != "" {
		s += ", fbtrace_id: " + e.FbtraceID
	}
	return s + ")"
}

// Codes Meta documents as temporary: the same request may succeed later.
const (
	CodeTemporarilyUnavailable = 2
	CodeAppRateLimit           = 4
	CodeWABARateLimit          = 80007
	CodeThroughputLimit        = 130429
	CodeUnknown                = 131000
	CodeServiceUnavailable     = 131016
	CodePairRateLimit          = 131056
	CodeServerUnavailable      = 133004
	CodeMaintenanceMode        = 2494100
)

// Codes for an access token that is expired or no longer valid.
const (
	CodeAuthUnknown  = 0
	CodeTokenExpired = 190
)

// CodeNotGraphError marks an *Error built from a non-2xx response whose body
// was not a Graph error envelope (e.g. a proxy error page).
const CodeNotGraphError = -1

// Codes callers commonly branch on.
const (
	CodeInvalidParameter         = 100
	CodeUndeliverable            = 131026
	CodeReEngagementRequired     = 131047
	CodeEcosystemEngagement      = 131049
	CodeMarketingOptOut          = 131050
	CodeTemplateParamMismatch    = 132000
	CodeTemplateNotFound         = 132001
	CodeTemplatePaused           = 132015
	CodeTemplateDisabled         = 132016
	CodeBusinessBlockedUser      = 130403
	CodeSpamRateLimit            = 131048
	CodeUnsupportedMessageType   = 131051
	CodePhoneNumberNotRegistered = 133010
)

// Temporary reports whether Meta documents the error as one to retry later:
// rate limits, throughput limits and service unavailability. 5xx responses
// without a recognised code are also treated as temporary.
func (e *Error) Temporary() bool {
	switch e.Code {
	case CodeTemporarilyUnavailable, CodeAppRateLimit, CodeWABARateLimit, CodeThroughputLimit,
		CodeUnknown, CodeServiceUnavailable, CodePairRateLimit, CodeServerUnavailable, CodeMaintenanceMode:
		return true
	}
	return e.HTTPStatus >= 500 || e.HTTPStatus == http.StatusTooManyRequests
}

// TokenInvalid reports whether the access token has expired or been
// invalidated, so a new token is needed before retrying.
func (e *Error) TokenInvalid() bool {
	switch e.Code {
	case CodeTokenExpired:
		return true
	case CodeAuthUnknown:
		return e.HTTPStatus == http.StatusUnauthorized || e.Type == "OAuthException"
	}
	return false
}

// AsError returns the *Error in err's chain, if any.
func AsError(err error) (*Error, bool) {
	var ge *Error
	if errors.As(err, &ge) {
		return ge, true
	}
	return nil, false
}

// IsTemporary reports whether err is a Graph error that may succeed on retry.
func IsTemporary(err error) bool {
	ge, ok := AsError(err)
	return ok && ge.Temporary()
}

// HasCode reports whether err is a Graph error with one of the given codes.
func HasCode(err error, codes ...int) bool {
	ge, ok := AsError(err)
	if !ok {
		return false
	}
	for _, c := range codes {
		if ge.Code == c {
			return true
		}
	}
	return false
}

// Parse builds an *Error from a non-2xx response body. A body that is not a
// Graph error envelope still yields an *Error carrying the HTTP status.
func Parse(status int, body []byte) *Error {
	var envelope struct {
		Error *Error `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error != nil {
		envelope.Error.HTTPStatus = status
		return envelope.Error
	}
	msg := string(body)
	if len(msg) > 512 {
		msg = msg[:512]
	}
	return &Error{HTTPStatus: status, Message: msg, Code: CodeNotGraphError}
}
