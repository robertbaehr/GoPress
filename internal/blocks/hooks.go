package blocks

import (
	"encoding/json"
	"strings"
	"sync"
)

// Relative positions for Block Hooks (wp-includes/blocks.php, P5.23).
const (
	HookBefore     = "before"
	HookAfter      = "after"
	HookFirstChild = "first_child"
	HookLastChild  = "last_child"
)

// hookedRegistry maps hooked block name → anchor block name → relative position.
// Mirrors WP_Block_Type::$block_hooks.
var (
	hookedMu       sync.RWMutex
	hookedRegistry = map[string]map[string]string{}
)

// RegisterHookedBlock registers a block to insert relative to an anchor (stub).
// position is one of before, after, first_child, last_child.
func RegisterHookedBlock(hookedName, anchorName, position string) {
	if hookedName == "" || anchorName == "" || position == "" {
		return
	}
	hookedMu.Lock()
	defer hookedMu.Unlock()
	if hookedRegistry[hookedName] == nil {
		hookedRegistry[hookedName] = map[string]string{}
	}
	hookedRegistry[hookedName][anchorName] = position
}

// ResetHookedBlocks clears the hooked-block registry (tests).
func ResetHookedBlocks() {
	hookedMu.Lock()
	defer hookedMu.Unlock()
	hookedRegistry = map[string]map[string]string{}
}

// GetHookedBlocks returns hooked block types grouped by anchor and relative position
// (get_hooked_blocks).
func GetHookedBlocks() map[string]map[string][]string {
	hookedMu.RLock()
	defer hookedMu.RUnlock()
	out := map[string]map[string][]string{}
	for hookedName, anchors := range hookedRegistry {
		for anchor, position := range anchors {
			if out[anchor] == nil {
				out[anchor] = map[string][]string{}
			}
			out[anchor][position] = append(out[anchor][position], hookedName)
		}
	}
	return out
}

// SerializeBlock serializes a parsed block to comment-delimited markup (serialize_block).
func SerializeBlock(block Block) string {
	name := blockName(block)
	if name == "" {
		return block.InnerHTML
	}
	inner := serializeInnerContent(block)
	attrs := serializeBlockAttrs(block.Attrs)
	if strings.TrimSpace(inner) == "" && len(block.InnerBlocks) == 0 {
		// Void / self-closing form when there is no inner markup.
		if attrs == "" {
			return "<!-- wp:" + name + " /-->"
		}
		return "<!-- wp:" + name + " " + attrs + " /-->"
	}
	if attrs == "" {
		return "<!-- wp:" + name + " -->" + inner + "<!-- /wp:" + name + " -->"
	}
	return "<!-- wp:" + name + " " + attrs + " -->" + inner + "<!-- /wp:" + name + " -->"
}

func serializeInnerContent(block Block) string {
	if len(block.InnerContent) == 0 {
		return block.InnerHTML
	}
	var b strings.Builder
	innerIdx := 0
	for _, chunk := range block.InnerContent {
		if s, ok := chunk.(string); ok {
			b.WriteString(s)
			continue
		}
		// nil marker → next inner block
		if innerIdx < len(block.InnerBlocks) {
			b.WriteString(SerializeBlock(block.InnerBlocks[innerIdx]))
			innerIdx++
		}
	}
	return b.String()
}

func serializeBlockAttrs(attrs map[string]any) string {
	if len(attrs) == 0 {
		return ""
	}
	raw, err := json.Marshal(attrs)
	if err != nil || string(raw) == "null" || string(raw) == "{}" {
		return ""
	}
	return string(raw)
}

// InsertHookedBlocks returns serialized markup for blocks hooked to the anchor
// at relative_position (insert_hooked_blocks stub).
func InsertHookedBlocks(anchor Block, position string, hooked map[string]map[string][]string) string {
	name := blockName(anchor)
	if name == "" || hooked == nil {
		return ""
	}
	byPos := hooked[name]
	if byPos == nil {
		return ""
	}
	types := byPos[position]
	if len(types) == 0 {
		return ""
	}
	ignored := ignoredHookedBlocks(anchor)
	var b strings.Builder
	for _, hookedType := range types {
		if ignored[hookedType] {
			continue
		}
		hb := Block{
			BlockName:    strPtr(hookedType),
			Attrs:        map[string]any{},
			InnerBlocks:  []Block{},
			InnerHTML:    "",
			InnerContent: []any{},
		}
		b.WriteString(SerializeBlock(hb))
	}
	return b.String()
}

func ignoredHookedBlocks(anchor Block) map[string]bool {
	out := map[string]bool{}
	if anchor.Attrs == nil {
		return out
	}
	meta, ok := anchor.Attrs["metadata"].(map[string]any)
	if !ok {
		return out
	}
	raw, ok := meta["ignoredHookedBlocks"]
	if !ok {
		return out
	}
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				out[s] = true
			}
		}
	case []string:
		for _, s := range v {
			out[s] = true
		}
	}
	return out
}

// ApplyBlockHooksToContent inserts registered hooked blocks into serialized
// content (apply_block_hooks_to_content stub). Supports before/after on
// top-level anchors and first_child/last_child on anchors with inner blocks.
func ApplyBlockHooksToContent(content string) string {
	hooked := GetHookedBlocks()
	if len(hooked) == 0 || content == "" {
		return content
	}
	parsed := Parse(content)
	var b strings.Builder
	for i := range parsed {
		b.WriteString(applyHooksToBlock(parsed[i], hooked))
	}
	return b.String()
}

func applyHooksToBlock(block Block, hooked map[string]map[string][]string) string {
	before := InsertHookedBlocks(block, HookBefore, hooked)
	after := InsertHookedBlocks(block, HookAfter, hooked)
	first := InsertHookedBlocks(block, HookFirstChild, hooked)
	last := InsertHookedBlocks(block, HookLastChild, hooked)

	if first == "" && last == "" {
		return before + SerializeBlock(block) + after
	}

	// Rebuild with first/last child injection around existing inner content.
	name := blockName(block)
	if name == "" {
		return before + block.InnerHTML + after
	}
	inner := first + serializeInnerContent(block) + last
	attrs := serializeBlockAttrs(block.Attrs)
	var mid string
	if attrs == "" {
		mid = "<!-- wp:" + name + " -->" + inner + "<!-- /wp:" + name + " -->"
	} else {
		mid = "<!-- wp:" + name + " " + attrs + " -->" + inner + "<!-- /wp:" + name + " -->"
	}
	return before + mid + after
}
