package business

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/gTahidi/wabo/internal/request_client"
	"github.com/gTahidi/wabo/internal/testutil"
	"github.com/gTahidi/wabo/pkg/graph"
)

func newTestBusinessClient(tr *testutil.Transport) *BusinessClient {
	return NewBusinessClient(&BusinessClientConfig{BusinessAccountId: "waba1", Requester: request_client.NewRequestClientWithHTTPClient("tok", tr.Client())})
}

func TestSubscribeAppContext(t *testing.T) {
	tr := &testutil.Transport{Body: `{"success":true}`}
	if err := newTestBusinessClient(tr).SubscribeAppContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	last := tr.Last()
	if last.Method != http.MethodPost || !strings.HasSuffix(last.URL, "/waba1/subscribed_apps") || last.Body != "" {
		t.Fatalf("unexpected request %s %s %q", last.Method, last.URL, last.Body)
	}

	tr = &testutil.Transport{Status: 403, Body: `{"error":{"message":"no","code":200}}`}
	if err := newTestBusinessClient(tr).SubscribeApp(); !graph.HasCode(err, 200) {
		t.Fatalf("want typed 200, got %v", err)
	}
	tr = &testutil.Transport{Body: `{"success":false}`}
	if err := newTestBusinessClient(tr).SubscribeApp(); err == nil {
		t.Fatal("want error for success=false")
	}
}

func TestFetchContextFields(t *testing.T) {
	tr := &testutil.Transport{Body: `{"id":"waba1","name":"Acme"}`}
	resp, err := newTestBusinessClient(tr).FetchContext(context.Background(), "id", "name")
	if err != nil || resp.Id != "waba1" || resp.Name != "Acme" {
		t.Fatalf("got %+v, %v", resp, err)
	}
	if last := tr.Last(); !strings.Contains(last.URL, "/waba1?fields=id%2Cname") {
		t.Fatalf("unexpected url %s", last.URL)
	}
}

func TestFetchBusinessPortfolioContext(t *testing.T) {
	tr := &testutil.Transport{Body: `{"id":"biz1","name":"Acme Ltd"}`}
	resp, err := newTestBusinessClient(tr).FetchBusinessPortfolioContext(context.Background(), "biz1")
	if err != nil || resp.Id != "biz1" || resp.Name != "Acme Ltd" {
		t.Fatalf("got %+v, %v", resp, err)
	}
	if last := tr.Last(); !strings.Contains(last.URL, "/biz1?fields=id%2Cname") {
		t.Fatalf("unexpected url %s", last.URL)
	}
	tr = &testutil.Transport{Status: 400, Body: `{"error":{"message":"bad","code":100}}`}
	if _, err := newTestBusinessClient(tr).FetchBusinessPortfolioContext(context.Background(), "biz1"); !graph.HasCode(err, graph.CodeInvalidParameter) {
		t.Fatalf("want typed 100, got %v", err)
	}
}
