package wp_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.150: RemotePost sends method POST + body; blocked when BlockExternal.
func TestRemotePostBlockExternalP2150(t *testing.T) {
	c := testBootQuery(t)
	if c.Config == nil {
		c.Config = &config.Config{}
	}
	c.Config.BlockExternal = true

	resp, err := wp.RemotePost(c, "https://example.com/", "a=1")
	if resp != nil {
		t.Fatalf("expected nil response")
	}
	he, ok := err.(*wp.HTTPError)
	if !ok {
		t.Fatalf("err type %T: %v", err, err)
	}
	if he.Code != "http_request_not_executed" {
		t.Fatalf("code=%q", he.Code)
	}
}

// P2.150: RemotePost posts body to a local test server.
func TestRemotePostSendsBodyP2150(t *testing.T) {
	c := testBootQuery(t)
	if c.Config == nil {
		c.Config = &config.Config{}
	}
	c.Config.BlockExternal = false

	var gotMethod, gotBody, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))
	defer srv.Close()

	resp, err := wp.RemotePost(c, srv.URL, "hello=world")
	if err != nil {
		t.Fatalf("RemotePost: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotBody != "hello=world" {
		t.Fatalf("body=%q", gotBody)
	}
	if !strings.Contains(gotCT, "application/x-www-form-urlencoded") {
		t.Fatalf("Content-Type=%q", gotCT)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if resp.Body != "created" {
		t.Fatalf("resp body=%q", resp.Body)
	}
}

// P2.150: SafeRemotePost rejects unsafe URLs before dialing.
func TestSafeRemotePostRejectsUnsafeP2150(t *testing.T) {
	c := testBootQuery(t)

	for _, bad := range []string{
		"",
		"javascript:alert(1)",
		"http://user:pass@example.com/",
		"http://127.0.0.2/x",
		"http://192.168.1.1/x",
	} {
		resp, err := wp.SafeRemotePost(c, bad, "x=1")
		if resp != nil {
			t.Fatalf("%q: expected nil response", bad)
		}
		he, ok := err.(*wp.HTTPError)
		if !ok {
			t.Fatalf("%q: err type %T: %v", bad, err, err)
		}
		if he.Code != "http_request_failed" {
			t.Fatalf("%q: code=%q", bad, he.Code)
		}
		if !strings.Contains(he.Message, "valid URL") && bad != "" {
			// empty URL fails earlier with "empty URL" via RemotePost after validation "".
			if bad == "" {
				continue
			}
			t.Fatalf("%q: message=%q", bad, he.Message)
		}
	}
}

// P2.150: SafeRemotePost still honors BlockExternal after validation.
func TestSafeRemotePostBlockExternalP2150(t *testing.T) {
	c := testBootQuery(t)
	if c.Config == nil {
		c.Config = &config.Config{}
	}
	c.Config.BlockExternal = true

	resp, err := wp.SafeRemotePost(c, "https://example.com/", "a=1")
	if resp != nil {
		t.Fatalf("expected nil response")
	}
	he, ok := err.(*wp.HTTPError)
	if !ok {
		t.Fatalf("err type %T: %v", err, err)
	}
	if he.Code != "http_request_not_executed" {
		t.Fatalf("code=%q", he.Code)
	}
}
