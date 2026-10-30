package oauth

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gTahidi/wabo/internal/testutil"
	"github.com/gTahidi/wabo/pkg/graph"
)

func TestExchangeCode(t *testing.T) {
	tr := &testutil.Transport{Body: `{"access_token":"EAAB","token_type":"bearer","expires_in":3600}`}
	token, err := ExchangeCode(context.Background(), ExchangeCodeParams{
		AppID: "app", AppSecret: "secret", Code: "c0de", RedirectURI: "https://x/cb", HTTPClient: tr.Client(), APIVersion: "v26.0",
	})
	if err != nil || token.AccessToken != "EAAB" {
		t.Fatalf("got %+v, %v", token, err)
	}
	now := time.Unix(0, 0)
	if !token.ExpiresAt(now).Equal(now.Add(time.Hour)) {
		t.Fatalf("expires at %v", token.ExpiresAt(now))
	}

	last := tr.Last()
	if last.Method != http.MethodPost || last.URL != "https://graph.facebook.com/v26.0/oauth/access_token" {
		t.Fatalf("unexpected request %s %s", last.Method, last.URL)
	}
	if last.Header.Get("Authorization") != "" {
		t.Fatal("exchange must not send a bearer token")
	}
	if ct := last.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		t.Fatalf("content type %q", ct)
	}
	form, _ := url.ParseQuery(last.Body)
	if form.Get("client_secret") != "secret" || form.Get("code") != "c0de" || form.Get("redirect_uri") != "https://x/cb" || form.Get("grant_type") != "authorization_code" {
		t.Fatalf("unexpected form %s", last.Body)
	}
}

func TestExchangeCodeErrors(t *testing.T) {
	tr := &testutil.Transport{Status: 400, Body: `{"error":{"message":"expired","type":"OAuthException","code":100,"error_subcode":36007}}`}
	_, err := ExchangeCode(context.Background(), ExchangeCodeParams{AppID: "a", AppSecret: "s", Code: "c", HTTPClient: tr.Client()})
	if ge, ok := graph.AsError(err); !ok || ge.ErrorSubcode != SubcodeCodeExpired {
		t.Fatalf("want subcode 36007, got %v", err)
	}

	tr = &testutil.Transport{Body: `{"token_type":"bearer"}`}
	if _, err := ExchangeCode(context.Background(), ExchangeCodeParams{AppID: "a", AppSecret: "s", Code: "c", HTTPClient: tr.Client()}); err == nil || !strings.Contains(err.Error(), "access_token") {
		t.Fatalf("want missing token error, got %v", err)
	}

	if _, err := ExchangeCode(context.Background(), ExchangeCodeParams{AppID: "a", Code: "c", HTTPClient: tr.Client()}); err == nil || len(tr.Requests) != 1 {
		t.Fatalf("want local error without a request, got %v", err)
	}
}
