package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFrontFeedRSS2Enclosure(t *testing.T) {
	c := testBoot(t)

	postID := int64(1) // Hello world
	encVal := "http://127.0.0.1:8082/wp-content/uploads/2024/06/sample-audio.mp3\n8612\naudio/mpeg\n"
	_, ok, err := c.Meta.AddMetadata(wp.MetaPost, postID, "enclosure", encVal, false)
	if err != nil || !ok {
		t.Fatalf("AddMetadata: ok=%v err=%v", ok, err)
	}
	t.Cleanup(func() {
		_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "enclosure", encVal)
	})

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<enclosure url="http://127.0.0.1:8082/wp-content/uploads/2024/06/sample-audio.mp3"`) {
		t.Fatalf("expected enclosure url: %s", truncateFeed(body, 400))
	}
	if !strings.Contains(body, `length="8612"`) || !strings.Contains(body, `type="audio/mpeg"`) {
		t.Fatalf("expected enclosure attrs: %s", truncateFeed(body, 400))
	}
}

func truncateFeed(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
