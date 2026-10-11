package wp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.153: RemoteRequest defaults method to GET; honors custom methods.
func TestRemoteRequestMethodP2153(t *testing.T) {
	c := testBootQuery(t)
	c.Config.BlockExternal = false

	var sawMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawMethod = r.Method
		w.Header().Set("Set-Cookie", "req=1; Path=/")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	resp, err := wp.RemoteRequest(c, srv.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if sawMethod != http.MethodGet {
		t.Fatalf("default method=%q", sawMethod)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	resp, err = wp.RemoteRequest(c, srv.URL, "put", "x=1")
	if err != nil {
		t.Fatal(err)
	}
	if sawMethod != http.MethodPut {
		t.Fatalf("put method=%q", sawMethod)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("put status=%d", resp.StatusCode)
	}
}

// P2.153: SafeRemoteRequest rejects unsafe URLs; BlockExternal still applies.
func TestSafeRemoteRequestP2153(t *testing.T) {
	c := testBootQuery(t)
	resp, err := wp.SafeRemoteRequest(c, "http://127.0.0.1/", "GET", "")
	if resp != nil || err == nil {
		t.Fatalf("expected unsafe reject: resp=%v err=%v", resp, err)
	}
	c.Config.BlockExternal = true
	resp, err = wp.SafeRemoteRequest(c, "https://example.com/", "GET", "")
	if resp != nil || err == nil {
		t.Fatalf("expected blocked: resp=%v err=%v", resp, err)
	}
	he, ok := err.(*wp.HTTPError)
	if !ok || he.Code != "http_request_not_executed" {
		t.Fatalf("err=%v", err)
	}
}
