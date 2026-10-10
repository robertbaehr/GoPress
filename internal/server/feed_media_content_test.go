package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFrontFeedRSS2MediaContent(t *testing.T) {
	c := testBoot(t)

	postID := int64(1) // Hello world
	prev := fmt.Sprint(c.Meta.GetMetadata(wp.MetaPost, postID, "_thumbnail_id", true))
	_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "_thumbnail_id", "")
	if _, _, err := c.Meta.AddMetadata(wp.MetaPost, postID, "_thumbnail_id", "5", true); err != nil {
		t.Fatalf("set thumbnail: %v", err)
	}
	encVal := "http://127.0.0.1:8082/wp-content/uploads/2024/06/sample-audio.mp3\n8612\naudio/mpeg\n"
	_, ok, err := c.Meta.AddMetadata(wp.MetaPost, postID, "enclosure", encVal, false)
	if err != nil || !ok {
		t.Fatalf("AddMetadata enclosure: ok=%v err=%v", ok, err)
	}
	t.Cleanup(func() {
		_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "enclosure", encVal)
		_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "_thumbnail_id", "")
		if prev != "" && prev != "<nil>" {
			_, _, _ = c.Meta.AddMetadata(wp.MetaPost, postID, "_thumbnail_id", prev, true)
		}
	})

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `xmlns:media="http://search.yahoo.com/mrss/"`) {
		t.Fatalf("missing media xmlns: %s", truncateFeed(body, 300))
	}
	if !strings.Contains(body, `<media:content url=`) {
		t.Fatalf("missing media:content: %s", truncateFeed(body, 500))
	}
	if !strings.Contains(body, `medium="image"`) {
		t.Fatalf("expected featured image media:content: %s", truncateFeed(body, 500))
	}
	if !strings.Contains(body, `type="image/jpeg"`) {
		t.Fatalf("expected image mime: %s", truncateFeed(body, 500))
	}
	if !strings.Contains(body, `medium="audio"`) || !strings.Contains(body, `fileSize="8612"`) {
		t.Fatalf("expected enclosure media:content: %s", truncateFeed(body, 500))
	}
}
