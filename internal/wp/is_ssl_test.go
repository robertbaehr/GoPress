package wp_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsSSLTLS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
	req.TLS = &tls.ConnectionState{}
	c := wp.NewContext(context.TODO())
	c.Req = req
	if !wp.IsSSL(c) {
		t.Fatal("expected TLS request to be SSL")
	}
}

func TestIsSSLHTTPSHeader(t *testing.T) {
	for _, val := range []string{"on", "ON", "1"} {
		req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
		req.Header.Set("HTTPS", val)
		c := wp.NewContext(context.TODO())
		c.Req = req
		if !wp.IsSSL(c) {
			t.Fatalf("HTTPS=%q should be SSL", val)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Header.Set("HTTPS", "off")
	c := wp.NewContext(context.TODO())
	c.Req = req
	if wp.IsSSL(c) {
		t.Fatal("HTTPS=off should not be SSL")
	}
}

func TestIsSSLPort443(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.test:443/", nil)
	c := wp.NewContext(context.TODO())
	c.Req = req
	if !wp.IsSSL(c) {
		t.Fatal("port 443 should be SSL")
	}
}

func TestIsSSLXForwardedProto(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	c := wp.NewContext(context.TODO())
	c.Req = req
	if !wp.IsSSL(c) {
		t.Fatal("X-Forwarded-Proto=https should be SSL")
	}
}

func TestIsSSLPlainHTTP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	c := wp.NewContext(context.TODO())
	c.Req = req
	if wp.IsSSL(c) {
		t.Fatal("plain HTTP should not be SSL")
	}
	if wp.IsSSL(nil) {
		t.Fatal("nil context should not be SSL")
	}
}
