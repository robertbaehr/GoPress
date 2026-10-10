// Block variations registry (P5.37) — subset of WP_Block_Type::$variations /
// registerBlockVariation server-side storage.
package wp

import (
	"strings"
	"sync"
)

// BlockVariation is a registered block variation (block.json / editor API subset).
type BlockVariation struct {
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Category    string         `json:"category,omitempty"`
	Icon        string         `json:"icon,omitempty"`
	IsDefault   bool           `json:"isDefault,omitempty"`
	Attributes  map[string]any `json:"attributes,omitempty"`
	Scope       []string       `json:"scope,omitempty"`
}

// BlockVariationRegistry stores variations keyed by block name then variation name.
type BlockVariationRegistry struct {
	mu         sync.RWMutex
	variations map[string]map[string]BlockVariation
}

// NewBlockVariationRegistry returns an empty registry.
func NewBlockVariationRegistry() *BlockVariationRegistry {
	return &BlockVariationRegistry{variations: map[string]map[string]BlockVariation{}}
}

// RegisterBlockVariation registers a variation for a block type
// (register_block_variation / WP_Block_Type::$variations subset).
func (r *BlockVariationRegistry) RegisterBlockVariation(blockName string, v BlockVariation) bool {
	if r == nil || blockName == "" || v.Name == "" {
		return false
	}
	if strings.Contains(v.Name, " ") {
		return false
	}
	if v.Title == "" {
		v.Title = v.Name
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.variations == nil {
		r.variations = map[string]map[string]BlockVariation{}
	}
	if r.variations[blockName] == nil {
		r.variations[blockName] = map[string]BlockVariation{}
	}
	r.variations[blockName][v.Name] = v
	return true
}

// UnregisterBlockVariation removes a variation for a block type.
func (r *BlockVariationRegistry) UnregisterBlockVariation(blockName, variationName string) bool {
	if r == nil || blockName == "" || variationName == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	m := r.variations[blockName]
	if m == nil {
		return false
	}
	if _, ok := m[variationName]; !ok {
		return false
	}
	delete(m, variationName)
	return true
}

// Get returns a registered variation, or false when missing.
func (r *BlockVariationRegistry) Get(blockName, variationName string) (BlockVariation, bool) {
	if r == nil {
		return BlockVariation{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := r.variations[blockName]
	if m == nil {
		return BlockVariation{}, false
	}
	v, ok := m[variationName]
	return v, ok
}

// IsRegistered reports whether a variation is registered for the block type.
func (r *BlockVariationRegistry) IsRegistered(blockName, variationName string) bool {
	_, ok := r.Get(blockName, variationName)
	return ok
}

// ForBlock returns variations registered for one block type.
func (r *BlockVariationRegistry) ForBlock(blockName string) []BlockVariation {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := r.variations[blockName]
	if len(m) == 0 {
		return nil
	}
	out := make([]BlockVariation, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// All returns a copy of all registered variations keyed by block name.
func (r *BlockVariationRegistry) All() map[string][]BlockVariation {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string][]BlockVariation, len(r.variations))
	for block, m := range r.variations {
		list := make([]BlockVariation, 0, len(m))
		for _, v := range m {
			list = append(list, v)
		}
		out[block] = list
	}
	return out
}

// RegisterBlockVariation is a Context helper for register_block_variation.
func (c *Context) RegisterBlockVariation(blockName string, v BlockVariation) bool {
	if c == nil {
		return false
	}
	if c.BlockVariations == nil {
		c.BlockVariations = NewBlockVariationRegistry()
	}
	return c.BlockVariations.RegisterBlockVariation(blockName, v)
}

// UnregisterBlockVariation is a Context helper for unregister_block_variation.
func (c *Context) UnregisterBlockVariation(blockName, variationName string) bool {
	if c == nil || c.BlockVariations == nil {
		return false
	}
	return c.BlockVariations.UnregisterBlockVariation(blockName, variationName)
}

// RegisterDefaultBlockVariations registers a small set of core block variations.
func RegisterDefaultBlockVariations(r *BlockVariationRegistry) {
	if r == nil {
		return
	}
	_ = r.RegisterBlockVariation("core/group", BlockVariation{
		Name:        "group",
		Title:       "Group",
		Description: "Gather blocks in a container.",
		IsDefault:   true,
		Scope:       []string{"block", "inserter", "transform"},
		Attributes:  map[string]any{"layout": map[string]any{"type": "constrained"}},
	})
	_ = r.RegisterBlockVariation("core/group", BlockVariation{
		Name:        "row",
		Title:       "Row",
		Description: "Arrange blocks horizontally.",
		Scope:       []string{"block", "inserter", "transform"},
		Attributes:  map[string]any{"layout": map[string]any{"type": "flex", "flexWrap": "nowrap"}},
	})
	_ = r.RegisterBlockVariation("core/group", BlockVariation{
		Name:        "stack",
		Title:       "Stack",
		Description: "Arrange blocks vertically.",
		Scope:       []string{"block", "inserter", "transform"},
		Attributes:  map[string]any{"layout": map[string]any{"type": "flex", "orientation": "vertical"}},
	})
	_ = r.RegisterBlockVariation("core/columns", BlockVariation{
		Name:       "two-columns-equal",
		Title:      "50 / 50",
		IsDefault:  true,
		Scope:      []string{"block"},
		Attributes: map[string]any{},
	})
}
