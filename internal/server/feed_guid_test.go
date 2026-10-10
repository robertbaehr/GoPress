package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFeedItemGUIDHelpers(t *testing.T) {
	got := feedItemGUID(nil, "http://example.test", 7, "")
	if got != "http://example.test/?p=7" {
		t.Fatalf("fallback: %q", got)
	}
	pretty := "http://example.test/2024/06/15/hello-world/"
	if feedItemGUID(nil, "http://example.test", 1, pretty) != pretty {
		t.Fatalf("stored pretty GUID: %q", feedItemGUID(nil, "http://example.test", 1, pretty))
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

func TestFeedItemGUIDGetTheGUIDFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	var sawID int64
	c.Hooks.AddFilter("get_the_guid", func(value any, args ...any) any {
		if len(args) > 0 {
			if id, ok := args[0].(int64); ok {
				sawID = id
			}
		}
		if s, ok := value.(string); ok {
			return s + "#filtered"
		}
		return "filtered"
	}, 10, 2)
	got := feedItemGUID(c, "http://example.test", 42, "http://example.test/?p=42")
	if got != "http://example.test/?p=42#filtered" {
		t.Fatalf("got %q", got)
	}
	if sawID != 42 {
		t.Fatalf("post id=%d", sawID)
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
