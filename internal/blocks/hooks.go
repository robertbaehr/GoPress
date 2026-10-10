package blocks

import (
	"encoding/json"
	"strings"
	"sync"
)

// Relative positions for Block Hooks (wp-includes/blocks.php, P5.23 / P5.41).
const (
	HookBefore     = "before"
	HookAfter      = "after"
	HookFirstChild = "first_child"
	HookLastChild  = "last_child"
)

// HookedBlocksCallback generates markup (and optionally mutates the anchor)
// for hooked blocks at a relative position (apply_block_hooks_to_content $callback).
type HookedBlocksCallback func(anchor *Block, position string, hooked map[string]map[string][]string) string

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
		if len(block.InnerBlocks) == 0 {
			return block.InnerHTML
		}
		var b strings.Builder
		for _, inner := range block.InnerBlocks {
			b.WriteString(SerializeBlock(inner))
		}
		return b.String()
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

// SetIgnoredHookedBlocksMetadata records hooked block types on the anchor's
// metadata.ignoredHookedBlocks (set_ignored_hooked_blocks_metadata, P5.41).
// Returns an empty string (WP returns ” — markup already emitted separately).
func SetIgnoredHookedBlocksMetadata(anchor *Block, position string, hooked map[string]map[string][]string) string {
	if anchor == nil || hooked == nil {
		return ""
	}
	name := blockName(*anchor)
	if name == "" {
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
	if anchor.Attrs == nil {
		anchor.Attrs = map[string]any{}
	}
	meta, _ := anchor.Attrs["metadata"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		anchor.Attrs["metadata"] = meta
	}
	prev := ignoredHookedBlocksList(*anchor)
	seen := map[string]bool{}
	merged := make([]string, 0, len(prev)+len(types))
	for _, s := range prev {
		if !seen[s] {
			seen[s] = true
			merged = append(merged, s)
		}
	}
	for _, s := range types {
		if !seen[s] {
			seen[s] = true
			merged = append(merged, s)
		}
	}
	anyList := make([]any, len(merged))
	for i, s := range merged {
		anyList[i] = s
	}
	meta["ignoredHookedBlocks"] = anyList
	return ""
}

// InsertHookedBlocksAndSetIgnored inserts hooked markup and records ignored
// metadata on the anchor (insert_hooked_blocks_and_set_ignored_hooked_blocks_metadata).
func InsertHookedBlocksAndSetIgnored(anchor *Block, position string, hooked map[string]map[string][]string) string {
	if anchor == nil {
		return ""
	}
	markup := InsertHookedBlocks(*anchor, position, hooked)
	_ = SetIgnoredHookedBlocksMetadata(anchor, position, hooked)
	return markup
}

func ignoredHookedBlocks(anchor Block) map[string]bool {
	out := map[string]bool{}
	for _, s := range ignoredHookedBlocksList(anchor) {
		out[s] = true
	}
	return out
}

func ignoredHookedBlocksList(anchor Block) []string {
	if anchor.Attrs == nil {
		return nil
	}
	meta, ok := anchor.Attrs["metadata"].(map[string]any)
	if !ok {
		return nil
	}
	raw, ok := meta["ignoredHookedBlocks"]
	if !ok {
		return nil
	}
	var out []string
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
	case []string:
		out = append(out, v...)
	}
	return out
}

// ApplyBlockHooksToContent inserts registered hooked blocks into serialized
// content (apply_block_hooks_to_content). Recurses into nested blocks (P5.41).
func ApplyBlockHooksToContent(content string) string {
	return ApplyBlockHooksToContentWith(content, insertHookedBlocksCallback)
}

// ApplyBlockHooksToContentWith runs the Block Hooks algorithm with a custom
// callback (default insert_hooked_blocks).
func ApplyBlockHooksToContentWith(content string, callback HookedBlocksCallback) string {
	hooked := GetHookedBlocks()
	if content == "" {
		return content
	}
	if callback == nil {
		callback = insertHookedBlocksCallback
	}
	if len(hooked) == 0 {
		return content
	}
	parsed := Parse(content)
	var b strings.Builder
	for i := range parsed {
		b.WriteString(applyHooksToBlock(&parsed[i], hooked, callback))
	}
	return b.String()
}

func insertHookedBlocksCallback(anchor *Block, position string, hooked map[string]map[string][]string) string {
	if anchor == nil {
		return ""
	}
	return InsertHookedBlocks(*anchor, position, hooked)
}

func applyHooksToBlock(block *Block, hooked map[string]map[string][]string, callback HookedBlocksCallback) string {
	if block == nil {
		return ""
	}
	before := callback(block, HookBefore, hooked)
	after := callback(block, HookAfter, hooked)
	first := callback(block, HookFirstChild, hooked)
	last := callback(block, HookLastChild, hooked)

	name := blockName(*block)
	if name == "" {
		return before + block.InnerHTML + after
	}

	inner := first + serializeInnerWithHooks(block, hooked, callback) + last
	attrs := serializeBlockAttrs(block.Attrs)
	if strings.TrimSpace(inner) == "" && len(block.InnerBlocks) == 0 && first == "" && last == "" {
		if attrs == "" {
			return before + "<!-- wp:" + name + " /-->" + after
		}
		return before + "<!-- wp:" + name + " " + attrs + " /-->" + after
	}
	var mid string
	if attrs == "" {
		mid = "<!-- wp:" + name + " -->" + inner + "<!-- /wp:" + name + " -->"
	} else {
		mid = "<!-- wp:" + name + " " + attrs + " -->" + inner + "<!-- /wp:" + name + " -->"
	}
	return before + mid + after
}

// serializeInnerWithHooks walks InnerContent / InnerBlocks and applies hooks
// recursively to nested anchors (traverse_and_serialize_blocks).
func serializeInnerWithHooks(block *Block, hooked map[string]map[string][]string, callback HookedBlocksCallback) string {
	if len(block.InnerContent) == 0 {
		if len(block.InnerBlocks) == 0 {
			return block.InnerHTML
		}
		var b strings.Builder
		for i := range block.InnerBlocks {
			b.WriteString(applyHooksToBlock(&block.InnerBlocks[i], hooked, callback))
		}
		return b.String()
	}
	var b strings.Builder
	innerIdx := 0
	for _, chunk := range block.InnerContent {
		if s, ok := chunk.(string); ok {
			b.WriteString(s)
			continue
		}
		if innerIdx < len(block.InnerBlocks) {
			b.WriteString(applyHooksToBlock(&block.InnerBlocks[innerIdx], hooked, callback))
			innerIdx++
		}
	}
	return b.String()
}
