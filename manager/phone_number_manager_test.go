package manager

import (
	"context"
	"strings"
	"testing"

	"github.com/gTahidi/wabo/internal/request_client"
	"github.com/gTahidi/wabo/internal/testutil"
)

func TestFetchAllContextSandboxAndFields(t *testing.T) {
	tr := &testutil.Transport{Body: `{"data":[{"id":"1","display_phone_number":"+1 555","code_verification_status":"VERIFIED"},{"id":"2","code_verification_status":{"status":"EXPIRED"}},{"id":"3","code_verification_status":null}]}`}
	pm := NewPhoneNumberManager(&PhoneNumberManagerConfig{BusinessAccountId: "waba1", Requester: request_client.NewRequestClientWithHTTPClient("tok", tr.Client())})

	edge, err := pm.FetchAllContext(context.Background(), FetchPhoneNumbersOptions{IncludeSandbox: true, Fields: []string{"id", "display_phone_number"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(edge.Data) != 3 || edge.Data[0].CodeVerificationStatus != WhatsappBusinessAccountPhoneNumberCodeVerificationStatusVerified ||
		edge.Data[1].CodeVerificationStatus != WhatsappBusinessAccountPhoneNumberCodeVerificationStatusExpired || edge.Data[2].CodeVerificationStatus != "" {
		t.Fatalf("unexpected data %+v", edge.Data)
	}
	last := tr.Last()
	if !strings.Contains(last.URL, "/waba1/phone_numbers?") || !strings.Contains(last.URL, "fields=id%2Cdisplay_phone_number") || strings.Contains(last.URL, "filtering") {
		t.Fatalf("unexpected url %s", last.URL)
	}

	if _, err := pm.FetchAll(false); err != nil {
		t.Fatal(err)
	}
	if last := tr.Last(); !strings.Contains(last.URL, "filtering=") || !strings.Contains(last.URL, "fields=id%2Caccount_mode") {
		t.Fatalf("want LIVE filter and default fields, got %s", last.URL)
	}
}

func TestFetchContextFields(t *testing.T) {
	tr := &testutil.Transport{Body: `{"id":"7","verified_name":"Acme"}`}
	pm := NewPhoneNumberManager(&PhoneNumberManagerConfig{Requester: request_client.NewRequestClientWithHTTPClient("tok", tr.Client())})
	phone, err := pm.FetchContext(context.Background(), "7", "id", "verified_name")
	if err != nil || phone.Id != "7" || phone.VerifiedName != "Acme" {
		t.Fatalf("got %+v, %v", phone, err)
	}
	if last := tr.Last(); !strings.HasSuffix(last.URL, "/7?fields=id%2Cverified_name") {
		t.Fatalf("unexpected url %s", last.URL)
	}
}
