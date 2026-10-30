package business

import (
	"github.com/gTahidi/wabo/internal/request_client"
	"github.com/gTahidi/wabo/internal/testutil"
	"testing"
)

func TestFetchDecodeError(t *testing.T) {
	for _, body := range []string{`{`, `{"id":123}`} {
		tr := &testutil.Transport{Body: body}
		client := NewBusinessClient(&BusinessClientConfig{BusinessAccountId: "b", Requester: request_client.NewRequestClientWithHTTPClient("token", tr.Client())})
		if response, err := client.Fetch(); err == nil || response != nil {
			t.Fatalf("body %q: response %+v error %v", body, response, err)
		}
	}
}
