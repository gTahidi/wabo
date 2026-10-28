package graph

import (
	"fmt"
	"testing"
)

func TestParseAndClassify(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		code      int
		temporary bool
		token     bool
	}{
		{"throughput", 400, `{"error":{"code":130429,"message":"(#130429) Rate limit hit"}}`, 130429, true, false},
		{"pair rate limit", 400, `{"error":{"code":131056}}`, 131056, true, false},
		{"service unavailable", 503, `{"error":{"code":131016}}`, 131016, true, false},
		{"token expired", 401, `{"error":{"code":190,"type":"OAuthException"}}`, 190, false, true},
		{"auth unknown", 401, `{"error":{"code":0,"type":"OAuthException"}}`, 0, false, true},
		{"re-engagement", 400, `{"error":{"code":131047}}`, 131047, false, false},
		{"template missing", 404, `{"error":{"code":132001}}`, 132001, false, false},
		{"html 500", 500, `oops`, CodeNotGraphError, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := Parse(tc.status, []byte(tc.body))
			if e.Code != tc.code || e.HTTPStatus != tc.status {
				t.Fatalf("got %+v", e)
			}
			if e.Temporary() != tc.temporary {
				t.Fatalf("Temporary() = %v", e.Temporary())
			}
			if e.TokenInvalid() != tc.token {
				t.Fatalf("TokenInvalid() = %v", e.TokenInvalid())
			}
		})
	}
}

func TestHelpersUnwrap(t *testing.T) {
	err := fmt.Errorf("send: %w", Parse(400, []byte(`{"error":{"code":130429}}`)))
	if !IsTemporary(err) || !HasCode(err, 1, CodeThroughputLimit) || HasCode(err, 100) {
		t.Fatal("helpers must see through wrapping")
	}
	if IsTemporary(fmt.Errorf("plain")) {
		t.Fatal("non-graph errors are not temporary")
	}
}

func TestErrorString(t *testing.T) {
	e := Parse(400, []byte(`{"error":{"code":100,"message":"(#100) Invalid parameter","error_data":{"details":"bad to"},"fbtrace_id":"X"}}`))
	want := "graph api 400: (#100) Invalid parameter (code 100, details: bad to, fbtrace_id: X)"
	if e.Error() != want {
		t.Fatalf("got %q", e.Error())
	}
}
