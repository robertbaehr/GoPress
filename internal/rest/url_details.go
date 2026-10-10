package rest

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// URLDetails is the wp-block-editor/v1/url-details response (P4.24 / P4.56).
// Remote HTML fetch stays omitted for SSRF safety; same-site URLs are enriched
// from options / posts, and callers may also parse HTML heads via ParseURLDetailsHTML.
type URLDetails struct {
	Title       string `json:"title"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

var (
	reHTMLTitle   = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</\s*title>`)
	reHTMLIcon = regexp.MustCompile(`(?is)<link[^>]+rel=["']?(?:icon|shortcut icon|icon shortcut)["']?[^>]*>`)
	reHrefAttr = regexp.MustCompile(`(?is)href=["']?([^"'\s>]+)`)
	reMetaOG      = regexp.MustCompile(`(?is)<meta\s[^>]*(?:property|name)=["']?(og:title|og:description|og:image|og:image:url|description)["']?[^>]*>`)
	reMetaContent = regexp.MustCompile(`(?is)content=["']([^"']*)["']`)
)

func (s *Server) serveURLDetails(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canEditPosts(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_view_url_details",
			"message": "Sorry, you are not allowed to process remote URLs.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_missing_callback_param",
			"message": "Missing parameter(s): url",
			"data": map[string]any{
				"status": 400,
				"params": []string{"url"},
			},
		})
		return
	}
	cleaned := wp.SanitizeURL(raw)
	if cleaned == "" || !isValidHTTPURL(cleaned) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_invalid_url",
			"message": "Invalid URL",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	cleaned = strings.TrimRight(cleaned, "/")
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusOK, enrichURLDetails(c, cleaned))
}

func isValidHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

func enrichURLDetails(c *wp.Context, target string) URLDetails {
	details := URLDetails{}
	if c == nil {
		return details
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	if home == "" || !sameOrigin(home, target) {
		// Remote URLs stay empty placeholders (no outbound fetch).
		return details
	}

	details.Title = optionString(c, "blogname", "")
	details.Description = optionString(c, "blogdescription", "")
	details.Icon = siteIconOrDefault(c, home)

	// Prefer singular post/page metadata when the path resolves.
	if post := resolveLocalURLPost(c, home, target); post != nil {
		if post.Title != "" {
			details.Title = post.Title
		}
		if excerpt := strings.TrimSpace(post.Excerpt); excerpt != "" {
			details.Description = excerpt
		} else if content := strings.TrimSpace(stripTagsShort(post.Content)); content != "" {
			if len(content) > 160 {
				content = content[:160]
			}
			details.Description = content
		}
	}
	return details
}

func siteIconOrDefault(c *wp.Context, home string) string {
	if icon := wp.GetSiteIconURL(c, 512, ""); icon != "" {
		return icon
	}
	if home == "" {
		return wp.DefaultSiteIconPath
	}
	return home + wp.DefaultSiteIconPath
}

func sameOrigin(home, target string) bool {
	h, err1 := url.Parse(home)
	t, err2 := url.Parse(target)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.EqualFold(h.Scheme, t.Scheme) && strings.EqualFold(h.Host, t.Host)
}

func resolveLocalURLPost(c *wp.Context, home, target string) *wp.Post {
	u, err := url.Parse(target)
	if err != nil {
		return nil
	}
	// Query permalinks: ?p=ID / ?page_id=ID
	if id := strings.TrimSpace(u.Query().Get("p")); id != "" {
		if post, err := wp.GetPostByID(c, parseInt64(id)); err == nil && post != nil {
			return post
		}
	}
	if id := strings.TrimSpace(u.Query().Get("page_id")); id != "" {
		if post, err := wp.GetPostByID(c, parseInt64(id)); err == nil && post != nil {
			return post
		}
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return nil
	}
	slug := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		slug = path[i+1:]
	}
	if slug == "" {
		return nil
	}
	if post, err := wp.GetPost(c, wp.QueryArgs{Name: slug, PostType: "post", PostStatus: "publish"}); err == nil && post != nil {
		return post
	}
	if page, err := wp.GetPost(c, wp.QueryArgs{PageName: slug, PostType: "page", PostStatus: "publish"}); err == nil && page != nil {
		return page
	}
	_ = home
	return nil
}

func parseInt64(s string) int64 {
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int64(r-'0')
	}
	return n
}

func stripTagsShort(s string) string {
	out := make([]rune, 0, len(s))
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			out = append(out, r)
		}
	}
	return strings.Join(strings.Fields(string(out)), " ")
}

// ParseURLDetailsHTML extracts title/icon/description/image from an HTML document
// head (subset of WP_REST_URL_Details_Controller parsers, P4.56).
func ParseURLDetailsHTML(html, pageURL string) URLDetails {
	d := URLDetails{}
	if m := reHTMLTitle.FindStringSubmatch(html); len(m) > 1 {
		d.Title = flattenMeta(m[1])
	}
	meta := map[string]string{}
	for _, m := range reMetaOG.FindAllStringSubmatch(html, -1) {
		if len(m) < 2 {
			continue
		}
		key := strings.ToLower(m[1])
		cm := reMetaContent.FindStringSubmatch(m[0])
		if len(cm) < 2 {
			continue
		}
		meta[key] = flattenMeta(cm[1])
	}
	if v := meta["og:title"]; v != "" && d.Title == "" {
		d.Title = v
	}
	if v := meta["og:description"]; v != "" {
		d.Description = v
	} else if v := meta["description"]; v != "" {
		d.Description = v
	}
	if v := meta["og:image"]; v != "" {
		d.Image = v
	} else if v := meta["og:image:url"]; v != "" {
		d.Image = v
	}
	if m := reHTMLIcon.FindString(html); m != "" {
		if hm := reHrefAttr.FindStringSubmatch(m); len(hm) > 1 {
			d.Icon = hm[1]
		}
	}
	_ = pageURL
	return d
}

func flattenMeta(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}
