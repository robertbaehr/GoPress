package rest

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// OEmbedResponse is a minimal oEmbed 1.0 JSON payload for a local post URL (P4.15).
type OEmbedResponse struct {
	Version      string `json:"version"`
	ProviderName string `json:"provider_name"`
	ProviderURL  string `json:"provider_url"`
	AuthorName   string `json:"author_name"`
	AuthorURL    string `json:"author_url"`
	Title        string `json:"title"`
	Type         string `json:"type"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	HTML         string `json:"html,omitempty"`
}

func (s *Server) serveOEmbed(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if rawURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "oembed_invalid_url",
			"message": "Invalid URL.",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	maxWidth := 600
	if v := r.URL.Query().Get("maxwidth"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxWidth = n
		}
	}
	if maxWidth < 200 {
		maxWidth = 200
	}
	if maxWidth > 600 {
		maxWidth = 600
	}

	post, err := resolveOEmbedPost(c, rawURL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "oembed_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "oembed_invalid_url",
			"message": "Not Found",
			"data":    map[string]any{"status": 404},
		})
		return
	}

	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	authorName := blogname
	authorURL := home
	if c != nil && c.Users != nil && post.Author > 0 {
		if u, err := c.Users.GetUserBy("id", post.Author); err == nil && u != nil {
			authorName = u.DisplayName
			if authorName == "" {
				authorName = u.Login
			}
			if home != "" {
				authorURL = home + "/author/" + u.Login + "/"
			}
		}
	}
	height := maxWidth * 9 / 16
	if height < 200 {
		height = 200
	}
	permalink := itemPermalink(c, home, *post)
	html := `<blockquote class="wp-embedded-content"><a href="` + permalink + `">` +
		htmlEscapeMinimal(post.Title) + `</a></blockquote>`

	writeJSON(w, http.StatusOK, OEmbedResponse{
		Version:      "1.0",
		ProviderName: blogname,
		ProviderURL:  home,
		AuthorName:   authorName,
		AuthorURL:    authorURL,
		Title:        post.Title,
		Type:         "rich",
		Width:        maxWidth,
		Height:       height,
		HTML:         html,
	})
}

func resolveOEmbedPost(c *wp.Context, rawURL string) (*wp.Post, error) {
	if c == nil || c.DB == nil {
		return nil, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, nil
	}
	q := u.Query()
	if p := q.Get("p"); p != "" {
		id, err := strconv.ParseInt(p, 10, 64)
		if err == nil && id > 0 {
			return wp.GetPost(c, wp.QueryArgs{P: id, PostStatus: "publish"})
		}
	}
	if pageID := q.Get("page_id"); pageID != "" {
		id, err := strconv.ParseInt(pageID, 10, 64)
		if err == nil && id > 0 {
			return wp.GetPost(c, wp.QueryArgs{P: id, PostType: "page", PostStatus: "publish"})
		}
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return nil, nil
	}
	parts := strings.Split(path, "/")
	slug := parts[len(parts)-1]
	if slug == "" {
		return nil, nil
	}
	// Try post then page by slug.
	post, err := wp.GetPost(c, wp.QueryArgs{Name: slug, PostType: "post", PostStatus: "publish"})
	if err != nil || post != nil {
		return post, err
	}
	return wp.GetPost(c, wp.QueryArgs{PageName: slug, PostType: "page", PostStatus: "publish"})
}

func htmlEscapeMinimal(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
