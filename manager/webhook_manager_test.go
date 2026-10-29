package manager

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gTahidi/wapi.go/internal/request_client"
	"github.com/gTahidi/wapi.go/pkg/events"
	"github.com/labstack/echo/v4"
)

const templateStatusPayload = `{"object":"whatsapp_business_account","entry":[{"id":"102290129340398","time":1739321024,"changes":[{"field":"message_template_status_update","value":{"event":"APPROVED","message_template_id":806312974732579,"message_template_name":"dd_quote_ready","message_template_language":"en","reason":"NONE"}}]}]}`

func sign(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func newTestWebhook(t *testing.T, appSecret string) *WebhookManager {
	t.Helper()
	wh := NewWebhook(&WebhookManagerConfig{
		Secret:       "verify-me",
		AppSecret:    appSecret,
		EventManager: NewEventManager(),
		Requester:    *request_client.NewRequestClient("tok"),
	})
	if wh == nil {
		t.Fatal("NewWebhook returned nil")
	}
	return wh
}

func post(wh *WebhookManager, body, signature string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	if signature != "" {
		req.Header.Set("X-Hub-Signature-256", signature)
	}
	rec := httptest.NewRecorder()
	_ = wh.PostRequestHandler(echo.New().NewContext(req, rec))
	return rec
}

func TestVerifyWebhookSignature(t *testing.T) {
	body := `{"a":1}`
	cases := []struct {
		name, header, secret string
		want                 bool
	}{
		{"valid", sign("s3cret", body), "s3cret", true},
		{"wrong secret", sign("other", body), "s3cret", false},
		{"missing prefix", strings.TrimPrefix(sign("s3cret", body), "sha256="), "s3cret", false},
		{"not hex", "sha256=zz", "s3cret", false},
		{"empty header", "", "s3cret", false},
		{"empty secret", sign("", body), "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := VerifyWebhookSignature([]byte(body), tc.header, tc.secret); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPostRequestHandlerRejectsBadSignature(t *testing.T) {
	wh := newTestWebhook(t, "app-secret")
	ch, _ := wh.EventManager.Subscribe(events.MessageTemplateStatusUpdateEventType)

	if rec := post(wh, templateStatusPayload, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing signature: status %d", rec.Code)
	}
	if rec := post(wh, templateStatusPayload, sign("wrong", templateStatusPayload)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong signature: status %d", rec.Code)
	}
	select {
	case ev := <-ch:
		t.Fatalf("rejected payload published %T", ev.Data)
	default:
	}
	if rec := post(wh, templateStatusPayload, sign("app-secret", templateStatusPayload)); rec.Code != http.StatusOK {
		t.Fatalf("valid signature: status %d body %s", rec.Code, rec.Body)
	}
}

// Meta sends entry.time as a number; it used to fail decoding, and the event
// was published under account_alerts instead of its own type.
func TestTemplateStatusEventDecodesAndUsesOwnType(t *testing.T) {
	wh := newTestWebhook(t, "")
	ch, _ := wh.EventManager.Subscribe(events.MessageTemplateStatusUpdateEventType)

	if rec := post(wh, templateStatusPayload, ""); rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body)
	}
	select {
	case ev := <-ch:
		e, ok := ev.Data.(*events.MessageTemplateStatusUpdateEvent)
		if !ok {
			t.Fatalf("got %T", ev.Data)
		}
		if e.MessageTemplateName != "dd_quote_ready" || e.Timestamp != "1739321024" || e.BusinessAccountId != "102290129340398" {
			t.Fatalf("unexpected event %+v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("no template status event published")
	}
}

func TestPostRequestHandlerInvalidJSONStops(t *testing.T) {
	wh := newTestWebhook(t, "")
	rec := post(wh, `{"entry":`, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
	if strings.Count(rec.Body.String(), "Invalid") != 1 {
		t.Fatalf("handler kept writing after rejecting: %q", rec.Body.String())
	}
}

func TestGetRequestHandler(t *testing.T) {
	wh := newTestWebhook(t, "")
	cases := []struct {
		query  string
		status int
		body   string
	}{
		{"hub.mode=subscribe&hub.verify_token=verify-me&hub.challenge=1158201444", 200, "1158201444"},
		{"hub.mode=subscribe&hub.verify_token=nope&hub.challenge=1", 400, "invalid token"},
		{"hub.mode=unsubscribe&hub.verify_token=verify-me&hub.challenge=1", 400, "invalid token"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/webhook?"+tc.query, nil)
		rec := httptest.NewRecorder()
		_ = wh.GetRequestHandler(echo.New().NewContext(req, rec))
		if rec.Code != tc.status || rec.Body.String() != tc.body {
			t.Fatalf("%s: got %d %q", tc.query, rec.Code, rec.Body.String())
		}
	}
}
