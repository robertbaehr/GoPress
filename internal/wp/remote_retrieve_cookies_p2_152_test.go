package wp_test

import (
	"net/http"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.152: RemoteRetrieveCookies / Cookie / CookieValue nil-safe helpers.
func TestRemoteRetrieveCookiesNilP2152(t *testing.T) {
	if got := wp.RemoteRetrieveCookies(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil cookies=%v", got)
	}
	if _, ok := wp.RemoteRetrieveCookie(nil, "x"); ok {
		t.Fatal("nil cookie")
	}
	if wp.RemoteRetrieveCookieValue(nil, "x") != "" {
		t.Fatal("nil cookie value")
	}
}

func TestRemoteRetrieveCookiesValuesP2152(t *testing.T) {
	resp := &wp.HTTPResponse{
		StatusCode: 200,
		Cookies: []wp.HTTPCookie{
			{Name: "wordpress_logged_in", Value: "abc"},
			{Name: "session", Value: "xyz"},
		},
		Headers: make(http.Header),
	}
	got := wp.RemoteRetrieveCookies(resp)
	if len(got) != 2 || got[0].Name != "wordpress_logged_in" || got[1].Value != "xyz" {
		t.Fatalf("cookies=%v", got)
	}
	// Copy isolation.
	got[0].Value = "mutated"
	if resp.Cookies[0].Value != "abc" {
		t.Fatal("retrieve should copy")
	}
	ck, ok := wp.RemoteRetrieveCookie(resp, "session")
	if !ok || ck.Value != "xyz" {
		t.Fatalf("cookie=%v ok=%v", ck, ok)
	}
	if _, ok := wp.RemoteRetrieveCookie(resp, "missing"); ok {
		t.Fatal("missing cookie")
	}
	if wp.RemoteRetrieveCookieValue(resp, "wordpress_logged_in") != "abc" {
		t.Fatal("cookie value")
	}
	if wp.RemoteRetrieveCookieValue(resp, "missing") != "" {
		t.Fatal("missing value")
	}
	if wp.RemoteRetrieveCookieValue(resp, "") != "" {
		t.Fatal("empty name")
	}
}

// P2.152: blocked RemoteGet → empty cookie retrieve helpers.
func TestRemoteRetrieveCookiesAfterFailedGetP2152(t *testing.T) {
	c := testBootQuery(t)
	c.Config.BlockExternal = true
	resp, err := wp.RemoteGet(c, "https://example.com/")
	if resp != nil || err == nil {
		t.Fatalf("expected blocked: resp=%v err=%v", resp, err)
	}
	if len(wp.RemoteRetrieveCookies(resp)) != 0 {
		t.Fatal("blocked cookies")
	}
	if wp.RemoteRetrieveCookieValue(resp, "x") != "" {
		t.Fatal("blocked cookie value")
	}
}
