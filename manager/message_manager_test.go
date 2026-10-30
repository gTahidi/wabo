package manager

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gTahidi/wabo/internal/request_client"
	"github.com/gTahidi/wabo/internal/testutil"
	"github.com/gTahidi/wabo/pkg/components"
	"github.com/gTahidi/wabo/pkg/graph"
)

func TestSendContextAuthenticationTemplate(t *testing.T) {
	tr := &testutil.Transport{Body: `{"messaging_product":"whatsapp","contacts":[{"input":"+254700000000","wa_id":"254700000000"}],"messages":[{"id":"wamid.HBgM"}]}`}
	mm := NewMessageManager(*request_client.NewRequestClientWithHTTPClient("tok", tr.Client()), "594266267109340")

	msg, err := components.NewAuthenticationTemplateMessage("code", "en", "482913")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := mm.SendContext(context.Background(), msg, "+254700000000")
	if err != nil {
		t.Fatal(err)
	}
	if resp.MessageID() != "wamid.HBgM" {
		t.Fatalf("message id %q", resp.MessageID())
	}

	last := tr.Last()
	if last.Method != http.MethodPost || !strings.HasSuffix(last.URL, "/594266267109340/messages") {
		t.Fatalf("unexpected request %s %s", last.Method, last.URL)
	}
	var payload struct {
		MessagingProduct string `json:"messaging_product"`
		To               string `json:"to"`
		Type             string `json:"type"`
		Template         struct {
			Name       string                `json:"name"`
			Language   struct{ Code string } `json:"language"`
			Components []struct {
				Type       string                        `json:"type"`
				SubType    string                        `json:"sub_type"`
				Index      json.Number                   `json:"index"`
				Parameters []struct{ Type, Text string } `json:"parameters"`
			} `json:"components"`
		} `json:"template"`
	}
	if err := json.Unmarshal([]byte(last.Body), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.MessagingProduct != "whatsapp" || payload.Type != "template" || payload.Template.Name != "code" || payload.Template.Language.Code != "en" {
		t.Fatalf("unexpected payload %s", last.Body)
	}
	c := payload.Template.Components
	if len(c) != 2 || c[0].Type != "body" || c[0].Parameters[0].Text != "482913" ||
		c[1].Type != "button" || c[1].SubType != "url" || c[1].Index.String() != "0" || c[1].Parameters[0].Text != "482913" {
		t.Fatalf("components do not match Meta's copy-code shape: %s", last.Body)
	}
}

func TestSendContextGraphError(t *testing.T) {
	tr := &testutil.Transport{Status: 400, Body: `{"error":{"message":"(#130429) Rate limit hit","type":"OAuthException","code":130429,"error_data":{"messaging_product":"whatsapp","details":"Cloud API message throughput has been reached."},"fbtrace_id":"A1"}}`}
	mm := NewMessageManager(*request_client.NewRequestClientWithHTTPClient("tok", tr.Client()), "1")
	msg, _ := components.NewTemplateMessage(&components.TemplateMessageConfigs{Name: "x", Language: "en"})
	_, err := mm.SendContext(context.Background(), msg, "+254700000000")
	if !graph.IsTemporary(err) || !graph.HasCode(err, graph.CodeThroughputLimit) {
		t.Fatalf("want temporary 130429, got %v", err)
	}
}

func TestSendErrorInSuccessBodyIsTyped(t *testing.T) {
	tr := &testutil.Transport{Body: `{"error":{"message":"bad","code":131047}}`}
	mm := NewMessageManager(*request_client.NewRequestClientWithHTTPClient("tok", tr.Client()), "1")
	msg, _ := components.NewTemplateMessage(&components.TemplateMessageConfigs{Name: "x", Language: "en"})
	_, err := mm.Send(msg, "+1")
	if !graph.HasCode(err, graph.CodeReEngagementRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestMigrateFromOtherBusinessAccountUsesPOSTAndDigits(t *testing.T) {
	tr := &testutil.Transport{Body: `{"migrated_templates":["1"],"failed_templates":{}}`}
	tm := NewTemplateManager(&TemplateManagerConfig{BusinessAccountId: "999", Requester: request_client.NewRequestClientWithHTTPClient("tok", tr.Client())})
	if _, err := tm.MigrateFromOtherBusinessAccount(2, 123456); err != nil {
		t.Fatal(err)
	}
	last := tr.Last()
	if last.Method != http.MethodPost || !strings.Contains(last.URL, "/999/migrate_message_templates?") ||
		!strings.Contains(last.URL, "page_number=2") || !strings.Contains(last.URL, "source_waba_id=123456") {
		t.Fatalf("unexpected request %s %s", last.Method, last.URL)
	}
}
