package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P7.152: attachment edit POST persists title/caption/description/alt.
func TestAttachmentEditSaveP7152(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/wp-admin/post.php?post=5&action=edit", nil)
	getReq.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	getRec := httptest.NewRecorder()
	gc := c.ForRequest(getRec, getReq)
	if !(&admin.Handler{}).Handle(getRec, getReq, gc, "/wp-admin/post.php") {
		t.Fatal("GET post.php not handled")
	}
	body := getRec.Body.String()
	nonce := extractInputValue(body, "_wpnonce")
	if nonce == "" {
		t.Fatalf("missing _wpnonce in %q", trunc(body, 400))
	}

	form := url.Values{}
	form.Set("action", "editattachment")
	form.Set("post_ID", "5")
	form.Set("post_type", "attachment")
	form.Set("_wpnonce", nonce)
	form.Set("post_title", "P7152 Attachment Title")
	form.Set("excerpt", "P7152 caption")
	form.Set("content", "P7152 description")
	form.Set("_wp_attachment_image_alt", "P7152 alt text")
	form.Set("save", "Update")

	postReq := httptest.NewRequest(http.MethodPost, "/wp-admin/post.php?post=5&action=edit", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	postRec := httptest.NewRecorder()
	pc := c.ForRequest(postRec, postReq)
	if !(&admin.Handler{}).Handle(postRec, postReq, pc, "/wp-admin/post.php") {
		t.Fatal("POST post.php not handled")
	}
	if postRec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", postRec.Code, postRec.Body.String())
	}
	out := postRec.Body.String()
	for _, want := range []string{
		"Media file updated.",
		`value="P7152 Attachment Title"`,
		`>P7152 caption</textarea>`,
		`>P7152 description</textarea>`,
		`value="P7152 alt text"`,
		`name="_wpnonce"`,
		`wp-header-end`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, trunc(out, 800))
		}
	}

	post, err := wp.GetPostByID(c, 5)
	if err != nil || post == nil {
		t.Fatalf("reload post: %v", err)
	}
	if post.Title != "P7152 Attachment Title" || post.Excerpt != "P7152 caption" || post.Content != "P7152 description" {
		t.Fatalf("persisted fields mismatch: title=%q excerpt=%q content=%q", post.Title, post.Excerpt, post.Content)
	}
	alt := asStringMeta(c, 5, "_wp_attachment_image_alt")
	if alt != "P7152 alt text" {
		t.Fatalf("alt meta=%q", alt)
	}
}

func extractInputValue(html, name string) string {
	// Prefer value=... name="..." then name=... value="..."
	for _, pattern := range []string{
		`name="` + name + `" value="`,
		`value="`,
	} {
		if pattern == `value="` {
			marker := `name="` + name + `"`
			i := strings.Index(html, marker)
			if i < 0 {
				return ""
			}
			start := i - 80
			if start < 0 {
				start = 0
			}
			end := i + 160
			if end > len(html) {
				end = len(html)
			}
			chunk := html[start:end]
			j := strings.Index(chunk, `value="`)
			if j < 0 {
				return ""
			}
			rest := chunk[j+len(`value="`):]
			k := strings.Index(rest, `"`)
			if k < 0 {
				return ""
			}
			return rest[:k]
		}
		marker := pattern
		i := strings.Index(html, marker)
		if i < 0 {
			continue
		}
		rest := html[i+len(marker):]
		k := strings.Index(rest, `"`)
		if k < 0 {
			continue
		}
		return rest[:k]
	}
	return ""
}

func asStringMeta(c *wp.Context, id int64, key string) string {
	if c == nil || c.Meta == nil {
		return ""
	}
	v := c.Meta.GetMetadata(wp.MetaPost, id, key, true)
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}
