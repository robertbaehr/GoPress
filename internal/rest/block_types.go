package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/blocks"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// BlockType is a minimal wp/v2 block-types object (P4.14 stub).
type BlockType struct {
	Name        string              `json:"name"`
	Title       string              `json:"title"`
	Category    string              `json:"category"`
	Description string              `json:"description,omitempty"`
	APIVersion  int                 `json:"api_version"`
	Attributes  map[string]any      `json:"attributes"`
	IsDynamic   bool                `json:"is_dynamic"`
	Variations  []wp.BlockVariation `json:"variations,omitempty"`
}

func knownBlockTypes() []BlockType {
	// P5.70: register core block metadata (name/title/category/description/is_dynamic)
	// for blocks GoPress already renders, matching block.json titles/categories.
	defs := []struct {
		name, title, category, description string
		dynamic                            bool
	}{
		{"core/paragraph", "Paragraph", "text", "Start with the basic building block of all narrative.", false},
		{"core/heading", "Heading", "text", "Introduce new sections and organize content to help visitors (and search engines) understand the structure of your content.", false},
		{"core/list", "List", "text", "An organized collection of items displayed in a specific order.", false},
		{"core/list-item", "List item", "text", "Create a list item.", false},
		{"core/image", "Image", "media", "Insert an image to make a visual statement.", false},
		{"core/quote", "Quote", "text", "Give quoted text visual emphasis.", false},
		{"core/code", "Code", "text", "Display code snippets that respect your spacing and tabs.", false},
		{"core/html", "Custom HTML", "widgets", "Add custom HTML code and preview it as you edit.", false},
		{"core/group", "Group", "design", "Gather blocks in a layout container.", false},
		{"core/columns", "Columns", "design", "Display content in multiple columns, with blocks added to each column.", false},
		{"core/column", "Column", "design", "A single column within a columns block.", false},
		{"core/buttons", "Buttons", "design", "Prompt visitors to take action with a group of button-style links.", false},
		{"core/button", "Button", "design", "Prompt visitors to take action with a button-style link.", false},
		{"core/separator", "Separator", "design", "Create a break between ideas or sections with a horizontal separator.", false},
		{"core/spacer", "Spacer", "design", "Add white space between blocks and customize its height.", false},
		{"core/embed", "Embed", "embed", "Add a block that displays content pulled from other sites.", false},
		{"core/template-part", "Template Part", "theme", "Edit the different global regions of your site, like the header, footer, sidebar, or create your own.", true},
		{"core/navigation", "Navigation", "theme", "A collection of blocks that allow visitors to get around your site.", true},
		{"core/navigation-link", "Custom Link", "design", "Add a page, link, or another item to your navigation.", false},
		{"core/home-link", "Home Link", "design", "Create a link that always points to the homepage of the site. Usually not necessary if there is already a site title link present in the header.", true},
		{"core/page-list", "Page List", "widgets", "Display a list of all pages.", true},
		{"core/details", "Details", "text", "Hide and show additional content.", false},
		{"core/footnotes", "Footnotes", "text", "Display footnotes added to the page.", true},
		{"core/freeform", "Classic", "text", "Use the classic WordPress editor.", false},
		{"core/missing", "Unsupported", "text", "Your site doesn’t include support for this block.", false},
		// P5.70 additions
		{"core/table", "Table", "text", "Create structured content in rows and columns to display information.", false},
		{"core/gallery", "Gallery", "media", "Display multiple images in a rich gallery.", false},
		{"core/cover", "Cover", "media", "Add an image or video with a text overlay.", false},
		{"core/media-text", "Media & Text", "media", "Set media and words side-by-side for a richer layout.", false},
		{"core/audio", "Audio", "media", "Embed a simple audio player.", false},
		{"core/video", "Video", "media", "Embed a video from your media library or upload a new one.", false},
		{"core/file", "File", "media", "Add a link to a downloadable file.", false},
		{"core/preformatted", "Preformatted", "text", "Add text that respects your spacing and tabs.", false},
		{"core/verse", "Poetry", "text", "Insert poetry. Use special spacing formats. Or quote song lyrics.", false},
		{"core/pullquote", "Pullquote", "text", "Give special visual emphasis to a quote from your text.", false},
		{"core/search", "Search", "widgets", "Help visitors find your content.", true},
		{"core/calendar", "Calendar", "widgets", "A calendar of your site’s posts.", true},
		{"core/archives", "Archives", "widgets", "Display a date archive of your posts.", true},
		{"core/latest-posts", "Latest Posts", "widgets", "Display a list of your most recent posts.", true},
		{"core/latest-comments", "Latest Comments", "widgets", "Display a list of your most recent comments.", true},
		{"core/tag-cloud", "Tag Cloud", "widgets", "A cloud of popular keywords, each sized by how often it appears.", true},
		{"core/read-more", "Read More", "theme", "Displays the link of a post, page, or any other content-type.", true},
		{"core/more", "More", "design", "Content before this block is shown in the excerpt on your archives.", false},
		{"core/nextpage", "Page Break", "design", "Separate your content into a multi-page experience.", false},
		{"core/social-links", "Social Icons", "widgets", "Display icons linking to your social profiles or sites.", false},
		{"core/social-link", "Social Icon", "widgets", "Display an icon linking to a social profile or site.", false},
		{"core/site-title", "Site Title", "theme", "Displays the name of this site.", true},
		{"core/site-tagline", "Site Tagline", "theme", "Describe in a few words what this site is about.", true},
		{"core/site-logo", "Site Logo", "theme", "Display an image to represent this site.", true},
		{"core/post-title", "Title", "theme", "Displays the title of a post, page, or any other content-type.", true},
		{"core/post-content", "Content", "theme", "Displays the contents of a post or page.", true},
		{"core/post-excerpt", "Excerpt", "theme", "Display the excerpt.", true},
		{"core/post-date", "Date", "theme", "Display a custom date.", true},
		{"core/post-featured-image", "Featured Image", "theme", "Display a post's featured image.", true},
		{"core/post-author", "Author", "theme", "Display post author details.", true},
		{"core/post-author-name", "Author Name", "theme", "The author name.", true},
		{"core/post-author-biography", "Author Biography", "theme", "The author biography.", true},
		{"core/avatar", "Avatar", "theme", "Add a user’s avatar.", true},
		{"core/post-terms", "Post Terms", "theme", "Post terms.", true},
		{"core/query", "Query Loop", "theme", "An advanced block that allows displaying post types based on specified parameters.", true},
		{"core/query-title", "Query Title", "theme", "Display the query title.", true},
		{"core/query-pagination", "Pagination", "theme", "Displays a paginated navigation to next/previous set of posts.", true},
		{"core/query-pagination-next", "Next Page", "theme", "Displays the next posts page link.", true},
		{"core/query-pagination-previous", "Previous Page", "theme", "Displays the previous posts page link.", true},
		{"core/query-pagination-numbers", "Page Numbers", "theme", "Displays a list of page numbers for pagination.", true},
		{"core/post-template", "Post Template", "theme", "Contains the block elements used to render a post.", true},
		{"core/query-no-results", "No Results", "theme", "Contains the block elements used to render content when no query results are found.", true},
		{"core/pattern", "Pattern Placeholder", "theme", "Show a block pattern.", true},
		{"core/block", "Pattern", "reusable", "Reuse this design across your site.", true},
		{"core/comments", "Comments", "theme", "An advanced block that allows displaying post comments using different visual configurations.", true},
		{"core/comment-template", "Comment Template", "design", "Contains the block elements used to display a comment.", true},
		{"core/comments-title", "Comments Title", "theme", "Displays a title with the number of comments.", true},
		{"core/post-comments-form", "Comments Form", "theme", "Display a post's comments form.", true},
		{"core/post-navigation-link", "Post Navigation Link", "theme", "Displays the next or previous post link that is adjacent to the current post.", true},
		{"core/terms-query", "Terms Query", "theme", "An advanced block that allows displaying taxonomy terms based on specified parameters.", true},
	}
	out := make([]BlockType, 0, len(defs))
	for _, d := range defs {
		out = append(out, BlockType{
			Name:        d.name,
			Title:       d.title,
			Category:    d.category,
			Description: d.description,
			APIVersion:  3,
			Attributes:  map[string]any{},
			IsDynamic:   d.dynamic,
		})
	}
	return out
}

func (s *Server) serveBlockTypes(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.serveBlockTypesFiltered(w, r, c, strings.TrimSpace(r.URL.Query().Get("namespace")))
}

// serveBlockTypesNamespace lists block types in a namespace path segment (P4.61).
func (s *Server) serveBlockTypesNamespace(w http.ResponseWriter, r *http.Request, c *wp.Context, ns string) {
	s.serveBlockTypesFiltered(w, r, c, strings.TrimSpace(ns))
}

func (s *Server) serveBlockTypesFiltered(w http.ResponseWriter, r *http.Request, c *wp.Context, ns string) {
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
			"code":    "rest_cannot_view",
			"message": "Sorry, you are not allowed to manage block types.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	all := attachBlockVariations(c, knownBlockTypes())
	if ns == "" {
		writeJSON(w, http.StatusOK, all)
		return
	}
	filtered := make([]BlockType, 0, len(all))
	prefix := ns + "/"
	for _, bt := range all {
		if strings.HasPrefix(bt.Name, prefix) {
			filtered = append(filtered, bt)
		}
	}
	writeJSON(w, http.StatusOK, filtered)
}

// serveBlockTypeItem serves GET /wp/v2/block-types/{namespace}/{name} (P4.61).
func (s *Server) serveBlockTypeItem(w http.ResponseWriter, r *http.Request, c *wp.Context, namespace, name string) {
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
			"code":    "rest_block_type_cannot_view",
			"message": "Sorry, you are not allowed to manage block types.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	if namespace == "" || name == "" || strings.Contains(namespace, "..") || strings.Contains(name, "..") {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_block_type_invalid",
			"message": "Invalid block type.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	full := namespace + "/" + name
	all := attachBlockVariations(c, knownBlockTypes())
	for _, bt := range all {
		if bt.Name == full {
			writeJSON(w, http.StatusOK, bt)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]any{
		"code":    "rest_block_type_invalid",
		"message": "Invalid block type.",
		"data":    map[string]any{"status": 404},
	})
}

// attachBlockVariations copies registered variations onto block-type payloads (P5.37).
func attachBlockVariations(c *wp.Context, types []BlockType) []BlockType {
	if c == nil || c.BlockVariations == nil {
		return types
	}
	out := make([]BlockType, len(types))
	copy(out, types)
	for i := range out {
		if vars := c.BlockVariations.ForBlock(out[i].Name); len(vars) > 0 {
			out[i].Variations = vars
		}
	}
	return out
}

func (s *Server) serveBlockRenderer(w http.ResponseWriter, r *http.Request, c *wp.Context, name string) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	attrs, postID, errMsg := parseBlockRendererRequest(r)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": errMsg,
			"data":    map[string]any{"status": 400},
		})
		return
	}
	if postID > 0 {
		p, err := wp.GetPost(c, wp.QueryArgs{P: postID})
		if err != nil || p == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"code":    "block_cannot_read",
				"message": "Sorry, you are not allowed to read blocks of this post.",
				"data":    map[string]any{"status": 401},
			})
			return
		}
		if !canEditPosts(c) {
			status := http.StatusUnauthorized
			if c != nil && c.CurrentUser != nil {
				status = http.StatusForbidden
			}
			writeJSON(w, status, map[string]any{
				"code":    "block_cannot_read",
				"message": "Sorry, you are not allowed to read blocks of this post.",
				"data":    map[string]any{"status": status},
			})
			return
		}
	} else if !canEditPosts(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "block_cannot_read",
			"message": "Sorry, you are not allowed to read blocks as this user.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	if !isKnownBlockName(name) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "block_invalid",
			"message": "Invalid block.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	rendered := renderBlockStub(name, attrs)
	writeJSON(w, http.StatusOK, map[string]any{"rendered": rendered})
}

// parseBlockRendererRequest reads attributes + post_id from GET query or POST JSON/form (P4.50).
func parseBlockRendererRequest(r *http.Request) (attrs map[string]any, postID int64, errMsg string) {
	attrs = map[string]any{}
	switch r.Method {
	case http.MethodGet:
		if raw := r.URL.Query().Get("attributes"); raw != "" {
			if err := json.Unmarshal([]byte(raw), &attrs); err != nil {
				return nil, 0, "Invalid parameter(s): attributes"
			}
		}
		if raw := strings.TrimSpace(r.URL.Query().Get("post_id")); raw != "" {
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || n < 0 {
				return nil, 0, "Invalid parameter(s): post_id"
			}
			postID = n
		}
	case http.MethodPost:
		ct := strings.ToLower(r.Header.Get("Content-Type"))
		if strings.Contains(ct, "application/json") || ct == "" {
			var body struct {
				Attributes map[string]any `json:"attributes"`
				PostID     any            `json:"post_id"`
			}
			if r.Body != nil {
				data, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
				_ = r.Body.Close()
				if len(bytes.TrimSpace(data)) > 0 {
					if err := json.Unmarshal(data, &body); err != nil {
						return nil, 0, "Invalid parameter(s): attributes"
					}
				}
			}
			if body.Attributes != nil {
				attrs = body.Attributes
			}
			postID = anyToInt64Loose(body.PostID)
		} else {
			_ = r.ParseForm()
			if raw := r.Form.Get("attributes"); raw != "" {
				if err := json.Unmarshal([]byte(raw), &attrs); err != nil {
					return nil, 0, "Invalid parameter(s): attributes"
				}
			}
			if raw := strings.TrimSpace(r.Form.Get("post_id")); raw != "" {
				n, err := strconv.ParseInt(raw, 10, 64)
				if err != nil || n < 0 {
					return nil, 0, "Invalid parameter(s): post_id"
				}
				postID = n
			}
		}
		// Query post_id also accepted on POST (WP REST).
		if postID == 0 {
			if raw := strings.TrimSpace(r.URL.Query().Get("post_id")); raw != "" {
				n, err := strconv.ParseInt(raw, 10, 64)
				if err != nil || n < 0 {
					return nil, 0, "Invalid parameter(s): post_id"
				}
				postID = n
			}
		}
	}
	return attrs, postID, ""
}

func anyToInt64Loose(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	case json.Number:
		n, _ := t.Int64()
		return n
	default:
		n, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(v)), 10, 64)
		return n
	}
}

func isKnownBlockName(name string) bool {
	for _, bt := range knownBlockTypes() {
		if bt.Name == name {
			return true
		}
	}
	return false
}

func renderBlockStub(name string, attrs map[string]any) string {
	attrJSON := ""
	if len(attrs) > 0 {
		b, err := json.Marshal(attrs)
		if err == nil {
			attrJSON = " " + string(b)
		}
	}
	// Prefer self-closing dynamic stubs; static blocks get a minimal inner wrapper.
	var serialized string
	switch name {
	case "core/paragraph":
		serialized = fmt.Sprintf("<!-- wp:paragraph%s -->\n<p></p>\n<!-- /wp:paragraph -->", attrJSON)
	case "core/heading":
		serialized = fmt.Sprintf("<!-- wp:heading%s -->\n<h2></h2>\n<!-- /wp:heading -->", attrJSON)
	case "core/separator":
		serialized = fmt.Sprintf("<!-- wp:separator%s /-->", attrJSON)
	case "core/spacer":
		serialized = fmt.Sprintf("<!-- wp:spacer%s /-->", attrJSON)
	default:
		short := name
		if strings.HasPrefix(name, "core/") {
			short = strings.TrimPrefix(name, "core/")
		}
		serialized = fmt.Sprintf("<!-- wp:%s%s /-->", short, attrJSON)
	}
	return blocks.Render(serialized)
}
