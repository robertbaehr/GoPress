// Package htmlapi is a minimal stub of WordPress HTML API (P1.12).
//
// This is not a full WP_HTML_Tag_Processor port. It provides a linear scanner
// that can find HTML tags and read attributes — enough for callers that need
// tag discovery without a complete HTML5 tokenizer.
package htmlapi

import (
	"strings"
	"unicode"
)

// TagQuery restricts NextTag matching (WP_HTML_Tag_Processor query subset).
type TagQuery struct {
	TagName    string // case-insensitive; empty matches any tag
	ClassName  string // optional class token that must be present
	TagClosers bool   // when true, also match closing tags (default: openers only)
}

// TagProcessor scans HTML for tags (WP_HTML_Tag_Processor subset).
type TagProcessor struct {
	html   string
	offset int

	tagName   string
	tagCloser bool
	attrs     map[string]string // lowercase name → decoded-ish raw value
	hasTag    bool
}

// NewTagProcessor creates a processor over html.
func NewTagProcessor(html string) *TagProcessor {
	return &TagProcessor{html: html}
}

// NextTag advances to the next matching tag. query may be nil (any opener),
// a tag name string via TagQuery{TagName: ...}, or a full TagQuery.
func (p *TagProcessor) NextTag(query *TagQuery) bool {
	if p == nil {
		return false
	}
	for {
		if !p.nextRawTag() {
			p.hasTag = false
			return false
		}
		if matchesQuery(p, query) {
			p.hasTag = true
			return true
		}
	}
}

// GetTag returns the uppercase tag name at the cursor, or "".
func (p *TagProcessor) GetTag() string {
	if p == nil || !p.hasTag {
		return ""
	}
	return strings.ToUpper(p.tagName)
}

// IsTagCloser reports whether the matched tag is a closing tag.
func (p *TagProcessor) IsTagCloser() bool {
	return p != nil && p.hasTag && p.tagCloser
}

// GetAttribute returns the attribute value for name (case-insensitive).
// Missing attributes return ("", false). Boolean attributes (name only)
// return ("", true).
func (p *TagProcessor) GetAttribute(name string) (string, bool) {
	if p == nil || p.tagName == "" || p.attrs == nil {
		return "", false
	}
	v, ok := p.attrs[strings.ToLower(name)]
	return v, ok
}

// HasClass reports whether the current tag has the given class token.
func (p *TagProcessor) HasClass(class string) bool {
	if p == nil || p.tagName == "" || class == "" {
		return false
	}
	val, ok := p.GetAttribute("class")
	if !ok || val == "" {
		return false
	}
	for _, tok := range strings.Fields(val) {
		if tok == class {
			return true
		}
	}
	return false
}

func matchesQuery(p *TagProcessor, q *TagQuery) bool {
	if q == nil {
		return !p.tagCloser
	}
	if p.tagCloser && !q.TagClosers {
		return false
	}
	if q.TagName != "" && !strings.EqualFold(p.tagName, q.TagName) {
		return false
	}
	if q.ClassName != "" && !p.HasClass(q.ClassName) {
		return false
	}
	return true
}

func (p *TagProcessor) nextRawTag() bool {
	html := p.html
	i := p.offset
	for i < len(html) {
		lt := strings.IndexByte(html[i:], '<')
		if lt < 0 {
			p.offset = len(html)
			return false
		}
		i += lt
		// Comments / bogus / doctype / PI: skip until '>'
		if i+1 < len(html) && (html[i+1] == '!' || html[i+1] == '?') {
			j := i + 2
			for j < len(html) && html[j] != '>' {
				j++
			}
			if j >= len(html) {
				p.offset = len(html)
				return false
			}
			i = j + 1
			continue
		}
		closer := false
		j := i + 1
		if j < len(html) && html[j] == '/' {
			closer = true
			j++
		}
		for j < len(html) && unicode.IsSpace(rune(html[j])) {
			j++
		}
		nameStart := j
		for j < len(html) {
			c := html[j]
			if unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || c == ':' || c == '-' {
				j++
				continue
			}
			break
		}
		if j == nameStart {
			i++
			continue
		}
		name := html[nameStart:j]
		// Scan attributes until '>'
		attrs := map[string]string{}
		for j < len(html) {
			for j < len(html) && unicode.IsSpace(rune(html[j])) {
				j++
			}
			if j >= len(html) {
				p.offset = len(html)
				return false
			}
			if html[j] == '>' {
				j++
				break
			}
			if html[j] == '/' {
				j++
				continue
			}
			attrStart := j
			for j < len(html) {
				c := html[j]
				if c == '=' || c == '>' || c == '/' || unicode.IsSpace(rune(c)) {
					break
				}
				j++
			}
			if j == attrStart {
				i++
				name = ""
				break
			}
			attrName := strings.ToLower(html[attrStart:j])
			for j < len(html) && unicode.IsSpace(rune(html[j])) {
				j++
			}
			if j < len(html) && html[j] == '=' {
				j++
				for j < len(html) && unicode.IsSpace(rune(html[j])) {
					j++
				}
				if j >= len(html) {
					p.offset = len(html)
					return false
				}
				switch html[j] {
				case '"':
					j++
					start := j
					for j < len(html) && html[j] != '"' {
						j++
					}
					attrs[attrName] = html[start:j]
					if j < len(html) {
						j++
					}
				case '\'':
					j++
					start := j
					for j < len(html) && html[j] != '\'' {
						j++
					}
					attrs[attrName] = html[start:j]
					if j < len(html) {
						j++
					}
				default:
					start := j
					for j < len(html) {
						c := html[j]
						if c == '>' || unicode.IsSpace(rune(c)) {
							break
						}
						j++
					}
					attrs[attrName] = html[start:j]
				}
			} else {
				attrs[attrName] = ""
			}
		}
		if name == "" {
			continue
		}
		p.offset = j
		p.tagName = name
		p.tagCloser = closer
		p.attrs = attrs
		return true
	}
	p.offset = len(html)
	return false
}
