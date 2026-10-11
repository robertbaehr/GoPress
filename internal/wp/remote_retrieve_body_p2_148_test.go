package wp_test

import (
	"net/http"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.148: RemoteRetrieveBody / ResponseCode / Header nil-safe helpers.
func TestRemoteRetrieveBodyNilP2148(t *testing.T) {
	if wp.RemoteRetrieveBody(nil) != "" {
		t.Fatal("nil body")
	}
	if wp.RemoteRetrieveResponseCode(nil) != 0 {
		t.Fatal("nil code")
	}
	if wp.RemoteRetrieveHeader(nil, "Content-Type") != "" {
		t.Fatal("nil header")
	}
	if wp.RemoteRetrieveHeaders(nil) != nil {
		t.Fatal("nil headers")
	}
}

func TestRemoteRetrieveBodyValuesP2148(t *testing.T) {
	h := make(http.Header)
	h.Set("Content-Type", "text/plain")
	h.Set("X-Test", "ok")
	resp := &wp.HTTPResponse{
		StatusCode: 201,
		Body:       "hello-body",
		Headers:    h,
	}
	if got := wp.RemoteRetrieveBody(resp); got != "hello-body" {
		t.Fatalf("body=%q", got)
	}
	if got := wp.RemoteRetrieveResponseCode(resp); got != 201 {
		t.Fatalf("code=%d", got)
	}
	if got := wp.RemoteRetrieveHeader(resp, "Content-Type"); got != "text/plain" {
		t.Fatalf("content-type=%q", got)
	}
	if got := wp.RemoteRetrieveHeader(resp, "x-test"); got != "ok" {
		t.Fatalf("x-test=%q", got)
	}
	if got := wp.RemoteRetrieveHeader(resp, "Missing"); got != "" {
		t.Fatalf("missing=%q", got)
	}
	if wp.RemoteRetrieveHeaders(resp) == nil {
		t.Fatal("headers nil")
	}
}

// P2.148: RemoteRetrieveBody works with HTTPError path (callers pass nil on err).
func TestRemoteRetrieveBodyAfterFailedGetP2148(t *testing.T) {
	c := testBootQuery(t)
	c.Config.BlockExternal = true
	resp, err := wp.RemoteGet(c, "https://example.com/")
	if resp != nil || err == nil {
		t.Fatalf("expected blocked: resp=%v err=%v", resp, err)
	}
	if wp.RemoteRetrieveBody(resp) != "" {
		t.Fatal("blocked retrieve body")
	}
	if wp.RemoteRetrieveResponseCode(resp) != 0 {
		t.Fatal("blocked retrieve code")
	}
}
