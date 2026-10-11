package wp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.160: GetServerProtocol maps request Proto; unknown → HTTP/1.0.
func TestGetServerProtocolP2160(t *testing.T) {
	if got := wp.GetServerProtocol(nil); got != "HTTP/1.0" {
		t.Fatalf("nil=%q", got)
	}
	c := testBootQuery(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Proto = "HTTP/1.1"
	req.ProtoMajor, req.ProtoMinor = 1, 1
	c = c.ForRequest(httptest.NewRecorder(), req)
	if got := wp.GetServerProtocol(c); got != "HTTP/1.1" {
		t.Fatalf("1.1=%q", got)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Proto = "HTTP/2.0"
	c2 := c.ForRequest(httptest.NewRecorder(), req2)
	if got := wp.GetServerProtocol(c2); got != "HTTP/2.0" {
		t.Fatalf("2.0=%q", got)
	}
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.Proto = "ICY"
	c3 := c.ForRequest(httptest.NewRecorder(), req3)
	if got := wp.GetServerProtocol(c3); got != "HTTP/1.0" {
		t.Fatalf("unknown=%q", got)
	}
}
