package wp_test

import (
	"net/http"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.149: RemoteRetrieveResponseCode / Message / CodeOK nil-safe helpers.
func TestRemoteRetrieveResponseCodeNilP2149(t *testing.T) {
	if wp.RemoteRetrieveResponseCode(nil) != 0 {
		t.Fatal("nil code")
	}
	if wp.RemoteRetrieveResponseMessage(nil) != "" {
		t.Fatal("nil message")
	}
	if wp.RemoteRetrieveResponseCodeOK(nil, 200) {
		t.Fatal("nil OK")
	}
}

func TestRemoteRetrieveResponseCodeValuesP2149(t *testing.T) {
	resp := &wp.HTTPResponse{
		StatusCode: 404,
		Status:     "Not Found",
		Body:       "missing",
	}
	if got := wp.RemoteRetrieveResponseCode(resp); got != 404 {
		t.Fatalf("code=%d", got)
	}
	if got := wp.RemoteRetrieveResponseMessage(resp); got != "Not Found" {
		t.Fatalf("message=%q", got)
	}
	if !wp.RemoteRetrieveResponseCodeOK(resp, 404) {
		t.Fatal("expected CodeOK 404")
	}
	if wp.RemoteRetrieveResponseCodeOK(resp, 200) {
		t.Fatal("unexpected CodeOK 200")
	}
}

// P2.149: empty Status falls back to http.StatusText(code).
func TestRemoteRetrieveResponseMessageFallbackP2149(t *testing.T) {
	resp := &wp.HTTPResponse{StatusCode: http.StatusCreated}
	if got := wp.RemoteRetrieveResponseMessage(resp); got != "Created" {
		t.Fatalf("message=%q", got)
	}
}

// P2.149: blocked RemoteGet → nil retrieve code/message.
func TestRemoteRetrieveResponseCodeAfterFailedGetP2149(t *testing.T) {
	c := testBootQuery(t)
	c.Config.BlockExternal = true
	resp, err := wp.RemoteGet(c, "https://example.com/")
	if resp != nil || err == nil {
		t.Fatalf("expected blocked: resp=%v err=%v", resp, err)
	}
	if wp.RemoteRetrieveResponseCode(resp) != 0 {
		t.Fatal("blocked retrieve code")
	}
	if wp.RemoteRetrieveResponseMessage(resp) != "" {
		t.Fatal("blocked retrieve message")
	}
	if wp.RemoteRetrieveResponseCodeOK(resp, 200) {
		t.Fatal("blocked CodeOK")
	}
}
