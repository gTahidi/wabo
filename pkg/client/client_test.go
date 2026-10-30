package wabo

import (
	"testing"
)

func TestNewPassesAccessTokenNotAccountID(t *testing.T) {
	c := New(&ClientConfig{BusinessAccountId: "waba-1", ApiAccessToken: "token-1", WebhookSecret: "s"})
	mc := c.NewMessagingClient("pn-1")
	if mc.ApiAccessToken != "token-1" {
		t.Fatalf("messaging client token = %q, want the access token", mc.ApiAccessToken)
	}
	if mc.BusinessAccountId != "waba-1" {
		t.Fatalf("business account id = %q", mc.BusinessAccountId)
	}
}

func TestAPIVersionPropagation(t *testing.T) {
	for _, version := range []string{"", "v24.0", "v26.0"} {
		c := New(&ClientConfig{APIVersion: version})
		want := version
		if want == "" {
			want = "v24.0"
		}
		if got := c.NewMessagingClient("phone").Requester.ApiVersion(); got != want {
			t.Fatalf("version %q want %q", got, want)
		}
	}
}
