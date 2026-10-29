// Package testutil stubs Graph API HTTP for tests.
package testutil

import (
	"io"
	"net/http"
	"strings"
	"sync"
)

// Recorded is one request seen by a Transport.
type Recorded struct {
	Method string
	URL    string
	Header http.Header
	Body   string
}

// Transport answers every request with Status and Body and records it.
type Transport struct {
	Status int
	Body   string
	// Handler, when set, replaces the fixed response.
	Handler func(*http.Request) (*http.Response, error)

	mu       sync.Mutex
	Requests []Recorded
}

func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	var body string
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
	}
	t.mu.Lock()
	t.Requests = append(t.Requests, Recorded{Method: r.Method, URL: r.URL.String(), Header: r.Header.Clone(), Body: body})
	t.mu.Unlock()
	if t.Handler != nil {
		return t.Handler(r)
	}
	status := t.Status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(t.Body)),
		Request:    r,
	}, nil
}

// Last returns the most recent request.
func (t *Transport) Last() Recorded {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.Requests) == 0 {
		return Recorded{}
	}
	return t.Requests[len(t.Requests)-1]
}

// Client returns an *http.Client using t.
func (t *Transport) Client() *http.Client { return &http.Client{Transport: t} }
