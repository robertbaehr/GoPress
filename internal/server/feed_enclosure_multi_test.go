package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFrontFeedMultipleEnclosures(t *testing.T) {
	c := testBoot(t)

	postID := int64(1) // Hello world
	enc1 := "http://127.0.0.1:8082/wp-content/uploads/2024/06/sample-audio.mp3\n8612\naudio/mpeg\n"
	enc2 := "http://127.0.0.1:8082/wp-content/uploads/2024/06/sample-video.mp4\r\n20480\r\nvideo/mp4 mp4\r\n"
	for _, enc := range []string{enc1, enc2} {
		_, ok, err := c.Meta.AddMetadata(wp.MetaPost, postID, "enclosure", enc, false)
		if err != nil || !ok {
			t.Fatalf("AddMetadata: ok=%v err=%v", ok, err)
		}
	}
	t.Cleanup(func() {
		_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "enclosure", enc1)
		_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "enclosure", enc2)
	})

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}

	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rss status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`<enclosure url="http://127.0.0.1:8082/wp-content/uploads/2024/06/sample-audio.mp3"`,
		`length="8612"`,
		`type="audio/mpeg"`,
		`<enclosure url="http://127.0.0.1:8082/wp-content/uploads/2024/06/sample-video.mp4"`,
		`length="20480"`,
		`type="video/mp4"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("rss missing %q: %s", want, truncateFeed(body, 500))
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/?feed=atom", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("atom status=%d", rec.Code)
	}
	body = rec.Body.String()
	for _, want := range []string{
		`rel="enclosure"`,
		`href="http://127.0.0.1:8082/wp-content/uploads/2024/06/sample-audio.mp3"`,
		`href="http://127.0.0.1:8082/wp-content/uploads/2024/06/sample-video.mp4"`,
		`type="audio/mpeg"`,
		`type="video/mp4"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("atom missing %q: %s", want, truncateFeed(body, 500))
		}
	}
}
