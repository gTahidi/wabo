package wapi

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
