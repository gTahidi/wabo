package request_client

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gTahidi/wabo/internal/testutil"
	"github.com/gTahidi/wabo/pkg/graph"
)

func TestExecuteEncodesQueryParams(t *testing.T) {
	tr := &testutil.Transport{Body: `{}`}
	rc := NewRequestClientWithHTTPClient("tok", tr.Client())
	req := rc.NewApiRequest("123/migrate_message_templates", http.MethodPost)
	req.AddQueryParam("page_number", "0")
	req.AddQueryParam("source_waba_id", "456")
	req.AddQueryParam("filter", "a b&c")
	if _, err := req.Execute(); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(tr.Last().URL)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != BASE_URL || u.Path != "/"+API_VERSION+"/123/migrate_message_templates" {
		t.Fatalf("unexpected url %s", u)
	}
	q := u.Query()
	if q.Get("page_number") != "0" || q.Get("source_waba_id") != "456" || q.Get("filter") != "a b&c" || len(q) != 3 {
		t.Fatalf("unexpected query %v", q)
	}
	if got := tr.Last().Header.Get("Authorization"); got != "Bearer tok" {
		t.Fatalf("authorization = %q", got)
	}
}

func TestExecuteReturnsGraphError(t *testing.T) {
	tr := &testutil.Transport{Status: 400, Body: `{"error":{"message":"(#131026) Message undeliverable","type":"OAuthException","code":131026,"error_data":{"messaging_product":"whatsapp","details":"Message Undeliverable."},"fbtrace_id":"Az8"}}`}
	rc := NewRequestClientWithHTTPClient("tok", tr.Client())
	_, err := rc.NewApiRequest("1/messages", http.MethodPost).Execute()
	ge, ok := graph.AsError(err)
	if !ok {
		t.Fatalf("want *graph.Error, got %T %v", err, err)
	}
	if ge.HTTPStatus != 400 || ge.Code != graph.CodeUndeliverable || ge.ErrorData.Details != "Message Undeliverable." || ge.FbtraceID != "Az8" {
		t.Fatalf("unexpected error %+v", ge)
	}
	if ge.Temporary() {
		t.Fatal("131026 must not be temporary")
	}
}

func TestExecuteNonGraphBody(t *testing.T) {
	tr := &testutil.Transport{Status: 502, Body: `<html>bad gateway</html>`}
	rc := NewRequestClientWithHTTPClient("tok", tr.Client())
	_, err := rc.NewApiRequest("x", http.MethodGet).Execute()
	ge, ok := graph.AsError(err)
	if !ok || ge.Code != graph.CodeNotGraphError || !ge.Temporary() {
		t.Fatalf("want temporary non-graph error, got %v", err)
	}
}

func TestExecuteContextHonoursDeadline(t *testing.T) {
	tr := &testutil.Transport{Handler: func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	}}
	rc := NewRequestClientWithHTTPClient("tok", tr.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := rc.NewApiRequest("x", http.MethodGet).ExecuteContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline exceeded, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("request outlived its context")
	}
}

func TestDefaultHTTPClientHasTimeout(t *testing.T) {
	if NewRequestClient("tok").HTTPClient().Timeout != DefaultTimeout {
		t.Fatal("default client must have a timeout")
	}
}

func TestVersionedURLs(t *testing.T) {
	for _, version := range []string{"v24.0", "v26.0"} {
		rc := NewRequestClientWithVersion("token", nil, version)
		if got, want := rc.url("phone/messages", nil), "https://graph.facebook.com/"+version+"/phone/messages"; got != want {
			t.Fatalf("URL %s want %s", got, want)
		}
	}
}
