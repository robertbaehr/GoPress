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
	defs := []struct {
		name, title, category string
		dynamic               bool
	}{
		{"core/paragraph", "Paragraph", "text", false},
		{"core/heading", "Heading", "text", false},
		{"core/list", "List", "text", false},
		{"core/list-item", "List item", "text", false},
		{"core/image", "Image", "media", false},
		{"core/quote", "Quote", "text", false},
		{"core/code", "Code", "text", false},
		{"core/html", "Custom HTML", "widgets", false},
		{"core/group", "Group", "design", false},
		{"core/columns", "Columns", "design", false},
		{"core/column", "Column", "design", false},
		{"core/buttons", "Buttons", "design", false},
		{"core/button", "Button", "design", false},
		{"core/separator", "Separator", "design", false},
		{"core/spacer", "Spacer", "design", false},
		{"core/embed", "Embed", "embed", false},
		{"core/template-part", "Template Part", "theme", true},
		{"core/navigation", "Navigation", "theme", true},
		{"core/navigation-link", "Custom Link", "design", false},
		{"core/page-list", "Page List", "widgets", true},
		{"core/details", "Details", "text", false},
		{"core/footnotes", "Footnotes", "text", true},
		{"core/freeform", "Classic", "text", false},
		{"core/missing", "Unsupported", "text", false},
	}
	out := make([]BlockType, 0, len(defs))
	for _, d := range defs {
		out = append(out, BlockType{
			Name:       d.name,
			Title:      d.title,
			Category:   d.category,
			APIVersion: 3,
			Attributes: map[string]any{},
			IsDynamic:  d.dynamic,
		})
	}
	return out
}

func (s *Server) serveBlockTypes(w http.ResponseWriter, r *http.Request, c *wp.Context) {
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
	ns := strings.TrimSpace(r.URL.Query().Get("namespace"))
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
