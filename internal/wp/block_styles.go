// Block styles registry (P5.26) — subset of WP_Block_Styles_Registry.
package wp

import (
	"strings"
	"sync"
)

// BlockStyle is a registered block style (register_block_style properties).
type BlockStyle struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	InlineStyle string `json:"inline_style,omitempty"`
	StyleHandle string `json:"style_handle,omitempty"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

// BlockStyleRegistry stores registered block styles keyed by block name then style name.
type BlockStyleRegistry struct {
	mu     sync.RWMutex
	styles map[string]map[string]BlockStyle
}

// NewBlockStyleRegistry returns an empty registry.
func NewBlockStyleRegistry() *BlockStyleRegistry {
	return &BlockStyleRegistry{styles: map[string]map[string]BlockStyle{}}
}

// RegisterBlockStyle registers a style for one or more block type names
// (register_block_style / WP_Block_Styles_Registry::register).
func (r *BlockStyleRegistry) RegisterBlockStyle(blockName string, style BlockStyle) bool {
	if r == nil || blockName == "" || style.Name == "" {
		return false
	}
	if strings.Contains(style.Name, " ") {
		return false
	}
	if style.Label == "" {
		style.Label = style.Name
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.styles == nil {
		r.styles = map[string]map[string]BlockStyle{}
	}
	if r.styles[blockName] == nil {
		r.styles[blockName] = map[string]BlockStyle{}
	}
	r.styles[blockName][style.Name] = style
	return true
}

// RegisterBlockStyles registers the same style across multiple block types.
func (r *BlockStyleRegistry) RegisterBlockStyles(blockNames []string, style BlockStyle) bool {
	ok := false
	for _, name := range blockNames {
		if r.RegisterBlockStyle(name, style) {
			ok = true
		}
	}
	return ok
}

// UnregisterBlockStyle removes a style for a block type.
func (r *BlockStyleRegistry) UnregisterBlockStyle(blockName, styleName string) bool {
	if r == nil || blockName == "" || styleName == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	m := r.styles[blockName]
	if m == nil {
		return false
	}
	if _, ok := m[styleName]; !ok {
		return false
	}
	delete(m, styleName)
	return true
}

// Get returns a registered style, or false when missing.
func (r *BlockStyleRegistry) Get(blockName, styleName string) (BlockStyle, bool) {
	if r == nil {
		return BlockStyle{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := r.styles[blockName]
	if m == nil {
		return BlockStyle{}, false
	}
	s, ok := m[styleName]
	return s, ok
}

// IsRegistered reports whether a style is registered for the block type.
func (r *BlockStyleRegistry) IsRegistered(blockName, styleName string) bool {
	_, ok := r.Get(blockName, styleName)
	return ok
}

// ForBlock returns styles registered for one block type.
func (r *BlockStyleRegistry) ForBlock(blockName string) []BlockStyle {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := r.styles[blockName]
	if len(m) == 0 {
		return nil
	}
	out := make([]BlockStyle, 0, len(m))
	for _, s := range m {
		out = append(out, s)
	}
	return out
}

// All returns a copy of all registered styles keyed by block name.
func (r *BlockStyleRegistry) All() map[string][]BlockStyle {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string][]BlockStyle, len(r.styles))
	for block, m := range r.styles {
		list := make([]BlockStyle, 0, len(m))
		for _, s := range m {
			list = append(list, s)
		}
		out[block] = list
	}
	return out
}

// RegisterBlockStyle is a Context helper for register_block_style.
func (c *Context) RegisterBlockStyle(blockName string, style BlockStyle) bool {
	if c == nil {
		return false
	}
	if c.BlockStyles == nil {
		c.BlockStyles = NewBlockStyleRegistry()
	}
	return c.BlockStyles.RegisterBlockStyle(blockName, style)
}

// UnregisterBlockStyle is a Context helper for unregister_block_style.
func (c *Context) UnregisterBlockStyle(blockName, styleName string) bool {
	if c == nil || c.BlockStyles == nil {
		return false
	}
	return c.BlockStyles.UnregisterBlockStyle(blockName, styleName)
}

// RegisterDefaultBlockStyles registers a small set of core block styles used by themes.
func RegisterDefaultBlockStyles(r *BlockStyleRegistry) {
	if r == nil {
		return
	}
	_ = r.RegisterBlockStyle("core/separator", BlockStyle{Name: "wide", Label: "Wide Line"})
	_ = r.RegisterBlockStyle("core/separator", BlockStyle{Name: "dots", Label: "Dots"})
	_ = r.RegisterBlockStyle("core/quote", BlockStyle{Name: "plain", Label: "Plain"})
	_ = r.RegisterBlockStyle("core/image", BlockStyle{Name: "rounded", Label: "Rounded"})
	_ = r.RegisterBlockStyle("core/button", BlockStyle{Name: "outline", Label: "Outline"})
	_ = r.RegisterBlockStyle("core/table", BlockStyle{Name: "stripes", Label: "Stripes"})
}
