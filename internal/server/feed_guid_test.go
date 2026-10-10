package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFeedItemGUIDHelpers(t *testing.T) {
	got := feedItemGUID("http://example.test", 7, "")
	if got != "http://example.test/?p=7" {
		t.Fatalf("fallback: %q", got)
	}
	pretty := "http://example.test/2024/06/15/hello-world/"
	if feedItemGUID("http://example.test", 1, pretty) != pretty {
		t.Fatalf("stored pretty GUID: %q", feedItemGUID("http://example.test", 1, pretty))
	}
	// WP hardcodes isPermaLink=false even when GUID == pretty permalink.
	if feedGUIDIsPermaLink(pretty, pretty) {
		t.Fatal("pretty permalink GUID must still be isPermaLink=false")
	}
	tag := rssGUIDTag(pretty, pretty)
	if !strings.Contains(tag, `isPermaLink="false"`) || !strings.Contains(tag, pretty) {
		t.Fatalf("rssGUIDTag: %q", tag)
	}
	if strings.Contains(tag, `isPermaLink="true"`) {
		t.Fatalf("must not emit true: %q", tag)
	}
}

func TestFeedRSSItemGUIDIsPermaLinkFalse(t *testing.T) {
	c := testBootFeedCats(t)
	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<guid isPermaLink="false">`) {
		t.Fatalf("expected isPermaLink=false: %q", body[:min(800, len(body))])
	}
	if strings.Contains(body, `isPermaLink="true"`) {
		t.Fatalf("WP never emits isPermaLink=true: %q", body[:min(800, len(body))])
	}
	// Fixture posts store pretty permalink URLs as GUID; still false.
	if !strings.Contains(body, `/?p=1</guid>`) && !strings.Contains(body, `<guid isPermaLink="false">http://`) {
		t.Fatalf("expected guid values in feed: %q", body[:min(800, len(body))])
	}
}
