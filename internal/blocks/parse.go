// Package blocks ports Gutenberg block serialization (P5.1+).
package blocks

import (
	"encoding/json"
	"strings"
	"unicode"
)

// Block is a parsed Gutenberg block (WP_Block_Parser_Block).
type Block struct {
	// BlockName is the block type (e.g. "core/paragraph"). Nil/empty for freeform HTML.
	BlockName *string `json:"blockName"`
	// Attrs are attributes from the block comment delimiter.
	Attrs map[string]any `json:"attrs"`
	// InnerBlocks are nested blocks.
	InnerBlocks []Block `json:"innerBlocks"`
	// InnerHTML is HTML inside delimiters with inner blocks removed.
	InnerHTML string `json:"innerHTML"`
	// InnerContent alternates HTML fragments and nil markers for inner blocks.
	InnerContent []any `json:"innerContent"`
}

// Parse parses serialized block comment markup into a block tree (parse_blocks).
func Parse(document string) []Block {
	p := &parser{document: document}
	return p.parse()
}

type parser struct {
	document string
	offset   int
	output   []Block
	stack    []frame
}

type frame struct {
	block            *Block
	tokenStart       int
	tokenLength      int
	prevOffset       int
	leadingHTMLStart *int
}

type token struct {
	typ    string // no-more-tokens | void-block | block-opener | block-closer
	name   string
	attrs  map[string]any
	start  int
	length int
}

func (p *parser) parse() []Block {
	p.offset = 0
	p.output = nil
	p.stack = nil
	for p.proceed() {
	}
	return p.output
}

func (p *parser) proceed() bool {
	tok := p.nextToken()
	stackDepth := len(p.stack)

	var leadingHTMLStart *int
	if tok.start > p.offset {
		o := p.offset
		leadingHTMLStart = &o
	}

	switch tok.typ {
	case "no-more-tokens":
		if stackDepth == 0 {
			p.addFreeform(-1)
			return false
		}
		if stackDepth == 1 {
			p.addBlockFromStack(nil)
			return false
		}
		for len(p.stack) > 0 {
			p.addBlockFromStack(nil)
		}
		return false

	case "void-block":
		if stackDepth == 0 {
			if leadingHTMLStart != nil {
				p.output = append(p.output, freeform(p.document[*leadingHTMLStart:tok.start]))
			}
			p.output = append(p.output, Block{
				BlockName:    strPtr(tok.name),
				Attrs:        emptyAttrs(tok.attrs),
				InnerBlocks:  []Block{},
				InnerHTML:    "",
				InnerContent: []any{},
			})
			p.offset = tok.start + tok.length
			return true
		}
		p.addInnerBlock(Block{
			BlockName:    strPtr(tok.name),
			Attrs:        emptyAttrs(tok.attrs),
			InnerBlocks:  []Block{},
			InnerHTML:    "",
			InnerContent: []any{},
		}, tok.start, tok.length, nil)
		p.offset = tok.start + tok.length
		return true

	case "block-opener":
		p.stack = append(p.stack, frame{
			block: &Block{
				BlockName:    strPtr(tok.name),
				Attrs:        emptyAttrs(tok.attrs),
				InnerBlocks:  []Block{},
				InnerHTML:    "",
				InnerContent: []any{},
			},
			tokenStart:       tok.start,
			tokenLength:      tok.length,
			prevOffset:       tok.start + tok.length,
			leadingHTMLStart: leadingHTMLStart,
		})
		p.offset = tok.start + tok.length
		return true

	case "block-closer":
		if stackDepth == 0 {
			p.addFreeform(-1)
			return false
		}
		if stackDepth == 1 {
			end := tok.start
			p.addBlockFromStack(&end)
			p.offset = tok.start + tok.length
			return true
		}
		stackTop := p.stack[len(p.stack)-1]
		p.stack = p.stack[:len(p.stack)-1]
		html := p.document[stackTop.prevOffset:tok.start]
		stackTop.block.InnerHTML += html
		stackTop.block.InnerContent = append(stackTop.block.InnerContent, html)
		last := tok.start + tok.length
		stackTop.prevOffset = last
		p.addInnerBlock(*stackTop.block, stackTop.tokenStart, stackTop.tokenLength, &last)
		p.offset = tok.start + tok.length
		return true

	default:
		p.addFreeform(-1)
		return false
	}
}

func (p *parser) nextToken() token {
	doc := p.document
	i := p.offset
	for {
		idx := strings.Index(doc[i:], "<!--")
		if idx < 0 {
			return token{typ: "no-more-tokens"}
		}
		start := i + idx
		pos := start + 4
		pos = skipWS(doc, pos)
		if pos >= len(doc) {
			return token{typ: "no-more-tokens"}
		}

		isCloser := false
		if doc[pos] == '/' {
			isCloser = true
			pos++
		}
		if pos+3 > len(doc) || doc[pos:pos+3] != "wp:" {
			i = start + 4
			continue
		}
		pos += 3

		ns, name, ok, npos := readBlockName(doc, pos)
		if !ok {
			i = start + 4
			continue
		}
		pos = npos
		blockName := ns + name

		// WP tokenizer requires whitespace after the block name.
		if pos >= len(doc) || !isHTMLSpace(doc[pos]) {
			i = start + 4
			continue
		}
		pos = skipWS(doc, pos)
		var attrs map[string]any
		hasAttrs := false
		if pos < len(doc) && doc[pos] == '{' {
			raw, end, aok := readJSONObject(doc, pos)
			if !aok {
				i = start + 4
				continue
			}
			hasAttrs = true
			_ = json.Unmarshal([]byte(raw), &attrs)
			if attrs == nil {
				attrs = map[string]any{}
			}
			pos = skipWS(doc, end)
		}

		isVoid := false
		if pos < len(doc) && doc[pos] == '/' {
			isVoid = true
			pos++
		}
		if pos+3 > len(doc) || doc[pos:pos+3] != "-->" {
			i = start + 4
			continue
		}
		pos += 3
		length := pos - start

		if isCloser && (isVoid || hasAttrs) {
			// Invalid but ignored, same as WP.
		}
		if isVoid {
			return token{typ: "void-block", name: blockName, attrs: attrs, start: start, length: length}
		}
		if isCloser {
			return token{typ: "block-closer", name: blockName, start: start, length: length}
		}
		return token{typ: "block-opener", name: blockName, attrs: attrs, start: start, length: length}
	}
}

func readBlockName(doc string, pos int) (ns, name string, ok bool, end int) {
	ns = "core/"
	// optional namespace: [a-z][a-z0-9_-]*/
	if pos < len(doc) && isNameStart(doc[pos]) {
		j := pos + 1
		for j < len(doc) && isNameChar(doc[j]) {
			j++
		}
		if j < len(doc) && doc[j] == '/' {
			ns = doc[pos : j+1]
			pos = j + 1
		}
	}
	if pos >= len(doc) || !isNameStart(doc[pos]) {
		return "", "", false, pos
	}
	j := pos + 1
	for j < len(doc) && isNameChar(doc[j]) {
		j++
	}
	return ns, doc[pos:j], true, j
}

func isNameStart(b byte) bool {
	return b >= 'a' && b <= 'z'
}

func isNameChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}

func isHTMLSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}

func skipWS(s string, i int) int {
	for i < len(s) {
		if isHTMLSpace(s[i]) || unicode.IsSpace(rune(s[i])) {
			i++
			continue
		}
		break
	}
	return i
}

// readJSONObject reads a JSON object starting at '{' with brace depth,
// respecting strings so '}' inside strings does not end the object.
func readJSONObject(s string, start int) (raw string, end int, ok bool) {
	if start >= len(s) || s[start] != '{' {
		return "", start, false
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1], i + 1, true
			}
		}
	}
	return "", start, false
}

func freeform(html string) Block {
	return Block{
		BlockName:    nil,
		Attrs:        map[string]any{},
		InnerBlocks:  []Block{},
		InnerHTML:    html,
		InnerContent: []any{html},
	}
}

func (p *parser) addFreeform(length int) {
	if length < 0 {
		length = len(p.document) - p.offset
	}
	if length == 0 {
		return
	}
	p.output = append(p.output, freeform(p.document[p.offset:p.offset+length]))
}

func (p *parser) addInnerBlock(block Block, tokenStart, tokenLength int, lastOffset *int) {
	parent := &p.stack[len(p.stack)-1]
	parent.block.InnerBlocks = append(parent.block.InnerBlocks, block)
	html := p.document[parent.prevOffset:tokenStart]
	if html != "" {
		parent.block.InnerHTML += html
		parent.block.InnerContent = append(parent.block.InnerContent, html)
	}
	parent.block.InnerContent = append(parent.block.InnerContent, nil)
	if lastOffset != nil {
		parent.prevOffset = *lastOffset
	} else {
		parent.prevOffset = tokenStart + tokenLength
	}
}

func (p *parser) addBlockFromStack(endOffset *int) {
	stackTop := p.stack[len(p.stack)-1]
	p.stack = p.stack[:len(p.stack)-1]
	prev := stackTop.prevOffset
	var html string
	if endOffset != nil {
		html = p.document[prev:*endOffset]
	} else {
		html = p.document[prev:]
	}
	if html != "" {
		stackTop.block.InnerHTML += html
		stackTop.block.InnerContent = append(stackTop.block.InnerContent, html)
	}
	if stackTop.leadingHTMLStart != nil {
		p.output = append(p.output, freeform(p.document[*stackTop.leadingHTMLStart:stackTop.tokenStart]))
	}
	p.output = append(p.output, *stackTop.block)
}

func strPtr(s string) *string { return &s }

func emptyAttrs(a map[string]any) map[string]any {
	if a == nil {
		return map[string]any{}
	}
	return a
}

// HasBlocks reports whether content contains a block comment opener.
func HasBlocks(content string) bool {
	return strings.Contains(content, "<!-- wp:")
}
