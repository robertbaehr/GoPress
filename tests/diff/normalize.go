// Package diff normalizes HTTP responses for GoPress differential comparison.
package diff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

// Compare modes for corpus entries.
const (
	CompareJSON    = "json"
	CompareHTML    = "html"
	CompareHeaders = "headers"
	CompareRaw     = "raw"
)

// Response is a captured HTTP response before normalization.
type Response struct {
	Status  int
	Headers http.Header
	Body    []byte
}

// CookieMeta is the comparable subset of a Set-Cookie header.
type CookieMeta struct {
	Name     string
	Path     string
	HTTPOnly bool
	Secure   bool
	SameSite string
}

// Normalized is the comparable form of a response.
type Normalized struct {
	Status  int
	Headers map[string]string
	Cookies []CookieMeta
	Body    string
}

var (
	nonceQueryRe  = regexp.MustCompile(`(?i)([?&](?:_wpnonce|_ajax_nonce|nonce)=)[a-zA-Z0-9_-]+`)
	nonceJSONRe   = regexp.MustCompile(`(?i)("(?:_wpnonce|_ajax_nonce|nonce)"\s*:\s*")[^"]+(")`)
	nonceAttrRe   = regexp.MustCompile(`(?i)((?:name|id)=["'](?:_wpnonce|_ajax_nonce|nonce)["'][^>]*value=["'])[^"']+(["'])`)
	nonceAttr2Re  = regexp.MustCompile(`(?i)(value=["'])[a-zA-Z0-9]{10}(["'][^>]*(?:name|id)=["'](?:_wpnonce|_ajax_nonce|nonce)["'])`)
	nonceHeaderRe = regexp.MustCompile(`(?i)(X-WP-Nonce:\s*)[a-zA-Z0-9_-]+`)
	// Standalone WP nonces are typically 10 URL-safe characters from wp_hash.
	wpNonceTokenRe = regexp.MustCompile(`\b[0-9a-f]{10}\b`)

	cookieAssignRe = regexp.MustCompile(`(?i)((?:wordpress(?:_logged_in|_sec)?_|wp-settings-(?:time-)?)[^=\s;]+=)[^;\s]+`)
	cookieHeaderRe = regexp.MustCompile(`(?i)((?:Cookie:\s*)?(?:wordpress(?:_logged_in|_sec)?_|wp-settings-(?:time-)?)[^=\s;]+=)[^;\s]+`)
)

// headerKeys are the response headers included in comparisons.
var headerKeys = []string{
	"Content-Type",
	"Location",
	"X-WP-Total",
	"X-WP-TotalPages",
	"Link",
	"Allow",
}

// Normalize applies corpus compare-mode rules and strips volatile values.
func Normalize(resp Response, compare string) (Normalized, error) {
	out := Normalized{
		Status:  resp.Status,
		Headers: map[string]string{},
		Cookies: parseSetCookies(resp.Headers.Values("Set-Cookie")),
	}
	for _, key := range headerKeys {
		vals := resp.Headers.Values(key)
		if len(vals) == 0 {
			continue
		}
		joined := strings.Join(vals, "\n")
		out.Headers[key] = scrubVolatile(joined)
	}

	body := scrubVolatile(string(resp.Body))
	switch strings.ToLower(compare) {
	case CompareJSON:
		canon, err := canonicalJSON([]byte(body))
		if err != nil {
			// Fall back to scrubbed text so diffs remain readable.
			out.Body = body
			return out, fmt.Errorf("json: %w", err)
		}
		out.Body = canon
	case CompareHTML:
		canon, err := canonicalHTML(body)
		if err != nil {
			out.Body = collapseWS(body)
			return out, fmt.Errorf("html: %w", err)
		}
		out.Body = canon
	case CompareHeaders:
		out.Body = ""
	case CompareRaw, "":
		out.Body = body
	default:
		out.Body = body
	}
	return out, nil
}

// Equal reports whether two normalized responses match.
func Equal(a, b Normalized) bool {
	if a.Status != b.Status {
		return false
	}
	if len(a.Headers) != len(b.Headers) {
		return false
	}
	for k, v := range a.Headers {
		if b.Headers[k] != v {
			return false
		}
	}
	if len(a.Cookies) != len(b.Cookies) {
		return false
	}
	ac := append([]CookieMeta(nil), a.Cookies...)
	bc := append([]CookieMeta(nil), b.Cookies...)
	sort.Slice(ac, func(i, j int) bool { return ac[i].Name < ac[j].Name })
	sort.Slice(bc, func(i, j int) bool { return bc[i].Name < bc[j].Name })
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return a.Body == b.Body
}

// DiffSummary returns a short human-readable mismatch description.
func DiffSummary(a, b Normalized) string {
	var parts []string
	if a.Status != b.Status {
		parts = append(parts, fmt.Sprintf("status %d≠%d", a.Status, b.Status))
	}
	for _, k := range headerKeys {
		av, aok := a.Headers[k]
		bv, bok := b.Headers[k]
		if aok != bok || av != bv {
			parts = append(parts, fmt.Sprintf("header %s", k))
		}
	}
	if len(a.Cookies) != len(b.Cookies) {
		parts = append(parts, "set-cookie meta")
	} else {
		ac := append([]CookieMeta(nil), a.Cookies...)
		bc := append([]CookieMeta(nil), b.Cookies...)
		sort.Slice(ac, func(i, j int) bool { return ac[i].Name < ac[j].Name })
		sort.Slice(bc, func(i, j int) bool { return bc[i].Name < bc[j].Name })
		for i := range ac {
			if ac[i] != bc[i] {
				parts = append(parts, "set-cookie meta")
				break
			}
		}
	}
	if a.Body != b.Body {
		parts = append(parts, fmt.Sprintf("body (%d vs %d bytes)", len(a.Body), len(b.Body)))
	}
	if len(parts) == 0 {
		return "match"
	}
	return strings.Join(parts, ", ")
}

func scrubVolatile(s string) string {
	s = nonceQueryRe.ReplaceAllString(s, `${1}NONCE`)
	s = nonceJSONRe.ReplaceAllString(s, `${1}NONCE${2}`)
	s = nonceAttrRe.ReplaceAllString(s, `${1}NONCE${2}`)
	s = nonceAttr2Re.ReplaceAllString(s, `${1}NONCE${2}`)
	s = nonceHeaderRe.ReplaceAllString(s, `${1}NONCE`)
	s = cookieAssignRe.ReplaceAllString(s, `${1}COOKIE`)
	s = cookieHeaderRe.ReplaceAllString(s, `${1}COOKIE`)
	// Remaining 10-char hex tokens in nonce-like contexts (localized scripts, etc.).
	s = wpNonceTokenRe.ReplaceAllString(s, "NONCE")
	return s
}

func canonicalJSON(raw []byte) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", nil
	}
	var v any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return "", err
	}
	// Reject trailing junk.
	if dec.More() {
		return "", fmt.Errorf("trailing data after JSON value")
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(sortJSON(v)); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func sortJSON(v any) any {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// Re-encode via ordered slice of pairs serialized as object by encoding/json
		// requires a map; Go's encoder sorts maps already since Go 1.20? Actually
		// encoding/json sorts map keys. Keep recursive sort of values.
		out := make(map[string]any, len(t))
		for _, k := range keys {
			out[k] = sortJSON(t[k])
		}
		return out
	case []any:
		for i := range t {
			t[i] = sortJSON(t[i])
		}
		return t
	default:
		return v
	}
}

func canonicalHTML(raw string) (string, error) {
	nodes, err := html.ParseFragment(strings.NewReader(raw), &html.Node{Type: html.ElementNode, Data: "body"})
	if err != nil {
		// Full document parse fallback.
		doc, err2 := html.Parse(strings.NewReader(raw))
		if err2 != nil {
			return "", err
		}
		return renderHTML(doc), nil
	}
	if len(nodes) == 0 {
		doc, err2 := html.Parse(strings.NewReader(raw))
		if err2 != nil {
			return "", err2
		}
		return renderHTML(doc), nil
	}
	// Prefer full-document parse for pages with doctype/html.
	if strings.Contains(strings.ToLower(raw), "<html") {
		doc, err2 := html.Parse(strings.NewReader(raw))
		if err2 != nil {
			return "", err2
		}
		return renderHTML(doc), nil
	}
	var b strings.Builder
	for _, n := range nodes {
		b.WriteString(renderHTML(n))
	}
	return b.String(), nil
}

func renderHTML(n *html.Node) string {
	normalizeTree(n)
	var buf bytes.Buffer
	if err := html.Render(&buf, n); err != nil {
		return ""
	}
	return buf.String()
}

func normalizeTree(n *html.Node) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode {
		sort.SliceStable(n.Attr, func(i, j int) bool {
			if n.Attr[i].Key == n.Attr[j].Key {
				return n.Attr[i].Namespace < n.Attr[j].Namespace
			}
			return n.Attr[i].Key < n.Attr[j].Key
		})
		for i := range n.Attr {
			n.Attr[i].Val = scrubVolatile(n.Attr[i].Val)
		}
	}
	if n.Type == html.TextNode {
		if strings.TrimSpace(n.Data) == "" {
			n.Data = " "
		} else {
			n.Data = scrubVolatile(collapseWS(n.Data))
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		normalizeTree(c)
	}
	// Drop whitespace-only text nodes between block-ish siblings by collapsing
	// consecutive space-only text nodes already handled; drop pure " " nodes
	// that are the only content noise when neighbors are elements.
	var keep []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && strings.TrimSpace(c.Data) == "" {
			// Keep a single space marker only when between two text-bearing nodes.
			continue
		}
		keep = append(keep, c)
	}
	if len(keep) > 0 || n.FirstChild != nil {
		n.FirstChild = nil
		n.LastChild = nil
		for _, c := range keep {
			c.Parent = nil
			c.PrevSibling = nil
			c.NextSibling = nil
			n.AppendChild(c)
		}
	}
}

func collapseWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func parseSetCookies(values []string) []CookieMeta {
	out := make([]CookieMeta, 0, len(values))
	for _, v := range values {
		meta := CookieMeta{}
		parts := strings.Split(v, ";")
		if len(parts) == 0 {
			continue
		}
		nv := strings.TrimSpace(parts[0])
		name, _, _ := strings.Cut(nv, "=")
		meta.Name = name
		for _, p := range parts[1:] {
			p = strings.TrimSpace(p)
			pl := strings.ToLower(p)
			switch {
			case pl == "httponly":
				meta.HTTPOnly = true
			case pl == "secure":
				meta.Secure = true
			case strings.HasPrefix(pl, "path="):
				meta.Path = p[len("path="):]
			case strings.HasPrefix(pl, "samesite="):
				meta.SameSite = p[len("samesite="):]
			}
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
