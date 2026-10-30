// Package oauth exchanges a Meta login authorization code, such as the one
// returned by Embedded Signup, for a business access token.
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gTahidi/wabo/internal/request_client"
)

// Meta error subcodes returned by the code exchange.
const (
	SubcodeCodeExpired         = 36007
	SubcodeRedirectURIMismatch = 36008
	SubcodeCodeAlreadyUsed     = 36009
)

// ExchangeCodeParams holds the app credentials and the code to exchange.
type ExchangeCodeParams struct {
	AppID     string
	AppSecret string
	Code      string
	// RedirectURI must match the one used to obtain Code; empty omits it.
	RedirectURI string

	// HTTPClient is used for the request; nil uses a shared client with a
	// 30 second timeout.
	HTTPClient *http.Client
	// APIVersion selects the Graph API version; empty uses the SDK default.
	APIVersion string
}

// Token is an access token returned by the exchange.
type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// ExpiresAt returns the expiry relative to now, or the zero time when Meta
// returned no expiry.
func (t *Token) ExpiresAt(now time.Time) time.Time {
	if t.ExpiresIn <= 0 {
		return time.Time{}
	}
	return now.Add(time.Duration(t.ExpiresIn) * time.Second)
}

// ExchangeCode exchanges an authorization code for an access token, bounded
// by ctx. A Graph API error is returned as a *graph.Error; branch on its
// ErrorSubcode using the Subcode constants.
func ExchangeCode(ctx context.Context, params ExchangeCodeParams) (*Token, error) {
	if params.AppID == "" || params.AppSecret == "" {
		return nil, errors.New("exchange code: app id and app secret are required")
	}
	if params.Code == "" {
		return nil, errors.New("exchange code: code is required")
	}

	form := url.Values{}
	form.Set("client_id", params.AppID)
	form.Set("client_secret", params.AppSecret)
	if params.RedirectURI != "" {
		form.Set("redirect_uri", params.RedirectURI)
	}
	form.Set("code", params.Code)
	form.Set("grant_type", "authorization_code")

	requester := request_client.NewRequestClientWithVersion("", params.HTTPClient, params.APIVersion)
	response, err := requester.RequestMultipartContext(ctx, http.MethodPost, "oauth/access_token",
		strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal([]byte(response), &token); err != nil {
		return nil, fmt.Errorf("exchange code: decode response: %w", err)
	}
	if token.AccessToken == "" {
		return nil, errors.New("exchange code: response missing access_token")
	}
	return &token, nil
}
