package messaging

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gTahidi/wabo/internal/request_client"
	"github.com/gTahidi/wabo/internal/testutil"
	"github.com/gTahidi/wabo/pkg/graph"
)

func newTestClient(tr *testutil.Transport) *MessagingClient {
	return &MessagingClient{
		PhoneNumberId: "594266267109340",
		Requester:     request_client.NewRequestClientWithHTTPClient("tok", tr.Client()),
	}
}

func TestRegisterContextSendsJSONBody(t *testing.T) {
	tr := &testutil.Transport{Body: `{"success":true}`}
	resp, err := newTestClient(tr).RegisterContext(context.Background(), "123456")
	if err != nil || !resp.Success {
		t.Fatalf("got %+v, %v", resp, err)
	}
	last := tr.Last()
	if last.Method != http.MethodPost || !strings.HasSuffix(last.URL, "/594266267109340/register") {
		t.Fatalf("unexpected request %s %s", last.Method, last.URL)
	}
	if strings.Contains(last.URL, "pin=") {
		t.Fatalf("pin leaked into the URL: %s", last.URL)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(last.Body), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["messaging_product"] != "whatsapp" || payload["pin"] != "123456" {
		t.Fatalf("unexpected payload %s", last.Body)
	}
	if ct := last.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type %q", ct)
	}
}

func TestRegisterOmitsEmptyPin(t *testing.T) {
	tr := &testutil.Transport{Body: `{"success":true}`}
	if _, err := newTestClient(tr).Register(""); err != nil {
		t.Fatal(err)
	}
	if body := tr.Last().Body; body != `{"messaging_product":"whatsapp"}` {
		t.Fatalf("unexpected payload %s", body)
	}
}

func TestDeregister(t *testing.T) {
	tr := &testutil.Transport{Body: `{"success":true}`}
	resp, err := newTestClient(tr).Deregister()
	if err != nil || !resp.Success {
		t.Fatalf("got %+v, %v", resp, err)
	}
	last := tr.Last()
	if last.Method != http.MethodPost || !strings.HasSuffix(last.URL, "/594266267109340/deregister") || last.Body != "" {
		t.Fatalf("unexpected request %s %s %q", last.Method, last.URL, last.Body)
	}
}

func TestPhoneNumberActionErrors(t *testing.T) {
	tr := &testutil.Transport{Status: 400, Body: `{"error":{"message":"not registered","code":133010}}`}
	if _, err := newTestClient(tr).DeregisterContext(context.Background()); !graph.HasCode(err, graph.CodePhoneNumberNotRegistered) {
		t.Fatalf("want typed 133010, got %v", err)
	}

	tr = &testutil.Transport{Body: `{"error":{"message":"bad","code":100}}`}
	_, err := newTestClient(tr).Register("123456")
	if ge, ok := graph.AsError(err); !ok || ge.Code != graph.CodeInvalidParameter || ge.HTTPStatus != http.StatusOK {
		t.Fatalf("want typed 100 from 2xx body, got %v", err)
	}

	tr = &testutil.Transport{Body: `{"success":false}`}
	if _, err := newTestClient(tr).Register("123456"); err == nil {
		t.Fatal("want error for success=false")
	}

	tr = &testutil.Transport{Body: `not json`}
	if _, err := newTestClient(tr).Register("123456"); err == nil {
		t.Fatal("want decode error")
	}
}

func TestRequestSMBAppDataSyncContext(t *testing.T) {
	tr := &testutil.Transport{Body: `{"messaging_product":"whatsapp","request_id":"r1"}`}
	resp, err := newTestClient(tr).RequestSMBAppDataSyncContext(context.Background(), SMBSyncTypeHistory)
	if err != nil || resp.RequestId != "r1" {
		t.Fatalf("got %+v, %v", resp, err)
	}
	last := tr.Last()
	if last.Method != http.MethodPost || !strings.HasSuffix(last.URL, "/594266267109340/smb_app_data") {
		t.Fatalf("unexpected request %s %s", last.Method, last.URL)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(last.Body), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["messaging_product"] != "whatsapp" || payload["sync_type"] != "HISTORY" {
		t.Fatalf("unexpected payload %s", last.Body)
	}

	tr = &testutil.Transport{Status: 400, Body: `{"error":{"message":"bad","code":100}}`}
	if _, err := newTestClient(tr).RequestSMBAppDataSyncContext(context.Background(), SMBSyncTypeAppState); !graph.HasCode(err, graph.CodeInvalidParameter) {
		t.Fatalf("want typed 100, got %v", err)
	}
}
