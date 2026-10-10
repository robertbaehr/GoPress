package blocks

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

// Supported block-binding attributes (wp-includes/block-bindings.php subset, P5.20/P5.39).
var bindingsSupportedAttrs = map[string][]string{
	"core/paragraph":          {"content"},
	"core/heading":            {"content"},
	"core/list-item":          {"content"},
	"core/button":             {"text", "url", "linkTarget", "rel"},
	"core/image":              {"id", "url", "alt", "title", "caption"},
	"core/post-date":          {"datetime"},
	"core/navigation-link":    {"url"},
	"core/navigation-submenu": {"url"},
}

// applyBindings resolves metadata.bindings sources and patches attrs / InnerHTML.
// Stub sources: core/post-meta and core/pattern-overrides.
func (e *Env) applyBindings(block Block) Block {
	e = e.ensure()
	name := blockName(block)
	supported := bindingsSupportedAttrs[name]
	if len(supported) == 0 || block.Attrs == nil {
		return block
	}
	meta, _ := block.Attrs["metadata"].(map[string]any)
	if meta == nil {
		return block
	}
	bindings, _ := meta["bindings"].(map[string]any)
	if len(bindings) == 0 {
		return block
	}

	// Expand __default pattern-overrides into per-attribute bindings.
	if def, ok := bindings["__default"].(map[string]any); ok {
		if src, _ := def["source"].(string); src == "core/pattern-overrides" {
			expanded := map[string]any{}
			for _, attr := range supported {
				if existing, ok := bindings[attr]; ok {
					expanded[attr] = existing
				} else {
					expanded[attr] = map[string]any{"source": "core/pattern-overrides"}
				}
			}
			bindings = expanded
		}
	}

	changed := false
	for _, attr := range supported {
		raw, ok := bindings[attr]
		if !ok {
			continue
		}
		binding, _ := raw.(map[string]any)
		if binding == nil {
			continue
		}
		source, _ := binding["source"].(string)
		args, _ := binding["args"].(map[string]any)
		val, ok := e.resolveBinding(source, args, attr, block)
		if !ok {
			continue
		}
		if block.Attrs == nil {
			block.Attrs = map[string]any{}
		}
		block.Attrs[attr] = val
		block.InnerHTML = replaceBoundHTML(block.InnerHTML, name, attr, val)
		changed = true
	}
	if !changed {
		return block
	}
	return block
}

func (e *Env) resolveBinding(source string, args map[string]any, attr string, block Block) (string, bool) {
	switch source {
	case "core/post-meta":
		key, _ := args["key"].(string)
		key = strings.TrimSpace(key)
		if key == "" || e.GetPostMeta == nil || e.currentPost == nil {
			return "", false
		}
		v := e.GetPostMeta(e.currentPost.ID, key)
		if v == "" {
			return "", false
		}
		return v, true
	case "core/post-data":
		return e.resolvePostDataBinding(args)
	case "core/term-data":
		return e.resolveTermDataBinding(args)
	case "twentytwentyfive/format":
		if e.PostFormat == nil || e.currentPost == nil {
			return "", false
		}
		slug := strings.TrimSpace(e.PostFormat(e.currentPost.ID))
		if slug == "" || slug == "standard" {
			return "", false
		}
		return slug, true
	case "core/pattern-overrides":
		if e.PatternOverrides == nil {
			return "", false
		}
		meta, _ := block.Attrs["metadata"].(map[string]any)
		if meta == nil {
			return "", false
		}
		name, _ := meta["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return "", false
		}
		override, ok := e.PatternOverrides[name]
		if !ok || override == nil {
			return "", false
		}
		v, ok := override[attr]
		if !ok || v == "" {
			return "", false
		}
		return v, true
	default:
		return "", false
	}
}

func (e *Env) resolvePostDataBinding(args map[string]any) (string, bool) {
	field, _ := args["field"].(string)
	if strings.TrimSpace(field) == "" {
		field, _ = args["key"].(string)
	}
	field = strings.TrimSpace(field)
	if field == "" || e.currentPost == nil {
		return "", false
	}
	switch field {
	case "date":
		if e.currentPost.Date == "" {
			return "", false
		}
		return e.currentPost.Date, true
	case "modified":
		mod := e.currentPost.Modified
		if mod == "" {
			return "", true // WP returns empty string when not later than publish.
		}
		return mod, true
	case "link":
		if e.currentPost.Link == "" {
			return "", false
		}
		return e.currentPost.Link, true
	default:
		return "", false
	}
}

func (e *Env) resolveTermDataBinding(args map[string]any) (string, bool) {
	field, _ := args["field"].(string)
	field = strings.TrimSpace(field)
	if field == "" || e.TermContext == nil || e.TermContext.ID == 0 || e.TermContext.Taxonomy == "" {
		return "", false
	}
	t := e.TermContext
	switch field {
	case "id":
		return strconv.FormatInt(t.ID, 10), true
	case "name":
		return t.Name, true
	case "link":
		if t.Link == "" {
			return "", false
		}
		return t.Link, true
	case "slug":
		return t.Slug, true
	case "description":
		return t.Description, true
	case "parent":
		return strconv.FormatInt(t.Parent, 10), true
	case "count":
		return strconv.FormatInt(t.Count, 10), true
	default:
		return "", false
	}
}

var (
	rePContent   = regexp.MustCompile(`(?is)(<p\b[^>]*>)(.*?)(</p>)`)
	reHContent   = regexp.MustCompile(`(?is)(<h[1-6]\b[^>]*>)(.*?)(</h[1-6]>)`)
	reLIContent  = regexp.MustCompile(`(?is)(<li\b[^>]*>)(.*?)(</li>)`)
	reButtonText = regexp.MustCompile(`(?is)(<a\b[^>]*>)(.*?)(</a>)`)
	reImgSrc     = regexp.MustCompile(`(?i)(\ssrc=")([^"]*)(")`)
	reImgAlt     = regexp.MustCompile(`(?i)(\salt=")([^"]*)(")`)
	reImgTitle   = regexp.MustCompile(`(?i)(\stitle=")([^"]*)(")`)
	reAHref      = regexp.MustCompile(`(?i)(\shref=")([^"]*)(")`)
)

func replaceBoundHTML(inner, blockType, attr, value string) string {
	if inner == "" {
		return inner
	}
	esc := html.EscapeString(value)
	switch blockType {
	case "core/paragraph":
		if attr == "content" {
			return rePContent.ReplaceAllString(inner, "${1}"+esc+"${3}")
		}
	case "core/heading":
		if attr == "content" {
			return reHContent.ReplaceAllString(inner, "${1}"+esc+"${3}")
		}
	case "core/list-item":
		if attr == "content" {
			return reLIContent.ReplaceAllString(inner, "${1}"+esc+"${3}")
		}
	case "core/button":
		switch attr {
		case "text":
			return reButtonText.ReplaceAllString(inner, "${1}"+esc+"${3}")
		case "url":
			return reAHref.ReplaceAllString(inner, "${1}"+esc+"${3}")
		}
	case "core/image":
		switch attr {
		case "url":
			return reImgSrc.ReplaceAllString(inner, "${1}"+esc+"${3}")
		case "alt":
			if reImgAlt.MatchString(inner) {
				return reImgAlt.ReplaceAllString(inner, "${1}"+esc+"${3}")
			}
			return strings.Replace(inner, "<img ", `<img alt="`+esc+`" `, 1)
		case "title":
			if reImgTitle.MatchString(inner) {
				return reImgTitle.ReplaceAllString(inner, "${1}"+esc+"${3}")
			}
		}
	}
	return inner
}
