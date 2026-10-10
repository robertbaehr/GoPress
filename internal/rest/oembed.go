package rest

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// OEmbedResponse is a minimal oEmbed 1.0 JSON payload for a local post URL (P4.15 / P3.99).
type OEmbedResponse struct {
	Version      string `json:"version" xml:"version"`
	ProviderName string `json:"provider_name" xml:"provider_name"`
	ProviderURL  string `json:"provider_url" xml:"provider_url"`
	AuthorName   string `json:"author_name" xml:"author_name"`
	AuthorURL    string `json:"author_url" xml:"author_url"`
	Title        string `json:"title" xml:"title"`
	Type         string `json:"type" xml:"type"`
	Width        int    `json:"width,omitempty" xml:"width,omitempty"`
	Height       int    `json:"height,omitempty" xml:"height,omitempty"`
	HTML         string `json:"html,omitempty" xml:"html,omitempty"`
}

func (s *Server) serveOEmbed(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
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
	// P3.99: honor maxheight when provided and smaller than calculated height.
	if v := r.URL.Query().Get("maxheight"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < height {
			height = n
			if height < 200 {
				height = 200
			}
		}
	}
	permalink := itemPermalink(c, home, *post)
	html := buildOEmbedRichHTML(permalink, post.Title, blogname, maxWidth, height)

	resp := OEmbedResponse{
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
	}
	resp = applyOEmbedResponseData(c, resp, post, maxWidth, height)

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "xml" {
		writeOEmbedXML(w, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// buildOEmbedRichHTML mirrors get_oembed_response_data_rich / get_post_embed_html
// without the inline wp-embed.js payload (P3.99).
func buildOEmbedRichHTML(permalink, title, blogname string, width, height int) string {
	secret := wp.GeneratePassword(10, false)
	embedURL := strings.TrimRight(permalink, "/") + "/embed/#?secret=" + secret
	titleAttr := "&#8220;" + htmlEscapeMinimal(title) + "&#8221; &#8212; " + htmlEscapeMinimal(blogname)
	var b strings.Builder
	b.WriteString(`<blockquote class="wp-embedded-content" data-secret="`)
	b.WriteString(htmlEscapeMinimal(secret))
	b.WriteString(`"><a href="`)
	b.WriteString(htmlEscapeMinimal(permalink))
	b.WriteString(`">`)
	b.WriteString(htmlEscapeMinimal(title))
	b.WriteString(`</a></blockquote>`)
	b.WriteString(`<iframe sandbox="allow-scripts" security="restricted" src="`)
	b.WriteString(htmlEscapeMinimal(embedURL))
	b.WriteString(`" width="`)
	b.WriteString(strconv.Itoa(width))
	b.WriteString(`" height="`)
	b.WriteString(strconv.Itoa(height))
	b.WriteString(`" title="`)
	b.WriteString(titleAttr)
	b.WriteString(`" data-secret="`)
	b.WriteString(htmlEscapeMinimal(secret))
	b.WriteString(`" frameborder="0" marginwidth="0" marginheight="0" scrolling="no" class="wp-embedded-content"></iframe>`)
	return b.String()
}

func applyOEmbedResponseData(c *wp.Context, resp OEmbedResponse, post *wp.Post, width, height int) OEmbedResponse {
	if c == nil || c.Hooks == nil || post == nil {
		return resp
	}
	filtered := c.Hooks.ApplyFilters("oembed_response_data", resp, post, width, height)
	if out, ok := filtered.(OEmbedResponse); ok {
		return out
	}
	return resp
}

func writeOEmbedXML(w http.ResponseWriter, resp OEmbedResponse) {
	type oembedXML struct {
		XMLName      xml.Name `xml:"oembed"`
		Version      string   `xml:"version"`
		ProviderName string   `xml:"provider_name"`
		ProviderURL  string   `xml:"provider_url"`
		AuthorName   string   `xml:"author_name"`
		AuthorURL    string   `xml:"author_url"`
		Title        string   `xml:"title"`
		Type         string   `xml:"type"`
		Width        int      `xml:"width,omitempty"`
		Height       int      `xml:"height,omitempty"`
		HTML         string   `xml:"html,omitempty"`
	}
	payload := oembedXML{
		Version:      resp.Version,
		ProviderName: resp.ProviderName,
		ProviderURL:  resp.ProviderURL,
		AuthorName:   resp.AuthorName,
		AuthorURL:    resp.AuthorURL,
		Title:        resp.Title,
		Type:         resp.Type,
		Width:        resp.Width,
		Height:       resp.Height,
		HTML:         resp.HTML,
	}
	w.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	if w.Header().Get("Allow") == "" {
		w.Header().Set("Allow", "GET")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(payload); err != nil {
		_, _ = fmt.Fprintf(w, "<!-- encode error: %v -->", err)
	}
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
