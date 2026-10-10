package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

func TestRSDXMLStub(t *testing.T) {
	s := &server.Server{}
	req := httptest.NewRequest(http.MethodGet, "/rsd.xml", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "xml") {
		t.Fatalf("Content-Type=%q", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`xmlns="http://archipelago.phrasewise.com/rsd"`,
		"<engineName>GoPress</engineName>",
		"<apis>",
		`name="WP-API"`,
		`apiLink="/wp-json/"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
	// Must not fall through to front 501.
	if strings.Contains(body, "not implemented") {
		t.Fatalf("unexpected front 501: %q", body)
	}
}

func TestRSDXMLDiscoveryWithContext(t *testing.T) {
	// P3.100: dynamic homePageLink + WP-API + Link header.
	c := testBoot(t)
	defer c.DB.Close()
	s := &server.Server{WP: c}
	req := httptest.NewRequest(http.MethodGet, "/rsd.xml", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	link := rec.Header().Get("Link")
	if !strings.Contains(link, `/wp-json/>; rel="https://api.w.org/"`) {
		t.Fatalf("Link=%q", link)
	}
	body := rec.Body.String()
	siteURL, _ := c.Options.GetOption("siteurl", "").(string)
	home, _ := c.Options.GetOption("home", siteURL).(string)
	home = strings.TrimRight(strings.TrimSpace(home), "/")
	if home == "" {
		home = strings.TrimRight(strings.TrimSpace(siteURL), "/")
	}
	if home == "" {
		t.Fatal("empty home")
	}
	for _, want := range []string{
		"<homePageLink>" + home + "</homePageLink>",
		`name="WP-API"`,
		`preferred="true"`,
		`apiLink="` + home + `/wp-json/"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
	// XML-RPC apis stay omitted (gone).
	for _, bad := range []string{`name="WordPress"`, `name="MetaWeblog"`, "xmlrpc.php"} {
		if strings.Contains(body, bad) {
			t.Fatalf("unexpected XML-RPC api %q in %q", bad, body)
		}
	}
}

func TestXMLRPCRsdStillGone(t *testing.T) {
	s := &server.Server{}
	req := httptest.NewRequest(http.MethodGet, "/xmlrpc.php?rsd", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}
