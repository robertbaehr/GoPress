// Block bindings sources registry (P5.39) — subset of
// WP_Block_Bindings_Registry / register_block_bindings_source.
package wp

import (
	"sort"
	"sync"
)

// BlockBindingsSource is a registered block bindings source.
type BlockBindingsSource struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	UsesContext []string `json:"uses_context"`
}

// BlockBindingsRegistry stores registered binding sources by name.
type BlockBindingsRegistry struct {
	mu      sync.RWMutex
	sources map[string]BlockBindingsSource
}

// NewBlockBindingsRegistry returns an empty registry.
func NewBlockBindingsRegistry() *BlockBindingsRegistry {
	return &BlockBindingsRegistry{sources: map[string]BlockBindingsSource{}}
}

// RegisterBlockBindingsSource adds or replaces a source (register_block_bindings_source).
func (r *BlockBindingsRegistry) RegisterBlockBindingsSource(name string, src BlockBindingsSource) bool {
	if r == nil || name == "" {
		return false
	}
	if src.Label == "" {
		src.Label = name
	}
	src.Name = name
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sources == nil {
		r.sources = map[string]BlockBindingsSource{}
	}
	r.sources[name] = src
	return true
}

// UnregisterBlockBindingsSource removes a source by name.
func (r *BlockBindingsRegistry) UnregisterBlockBindingsSource(name string) bool {
	if r == nil || name == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sources[name]; !ok {
		return false
	}
	delete(r.sources, name)
	return true
}

// Get returns a source by name.
func (r *BlockBindingsRegistry) Get(name string) (BlockBindingsSource, bool) {
	if r == nil {
		return BlockBindingsSource{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sources[name]
	return s, ok
}

// All returns registered sources sorted by name.
func (r *BlockBindingsRegistry) All() []BlockBindingsSource {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]BlockBindingsSource, 0, len(r.sources))
	for _, s := range r.sources {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Map returns sources keyed by name (golden / dump shape).
func (r *BlockBindingsRegistry) Map() map[string]BlockBindingsSource {
	if r == nil {
		return map[string]BlockBindingsSource{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]BlockBindingsSource, len(r.sources))
	for k, v := range r.sources {
		out[k] = v
	}
	return out
}

// RegisterDefaultBlockBindingsSources registers core sources from
// wp-includes/block-bindings/*.php plus Twenty Twenty-Five format when active.
func RegisterDefaultBlockBindingsSources(r *BlockBindingsRegistry, stylesheet string) {
	if r == nil {
		return
	}
	core := []BlockBindingsSource{
		{Name: "core/pattern-overrides", Label: "Pattern Overrides", UsesContext: []string{"pattern/overrides"}},
		{Name: "core/post-data", Label: "Post Data", UsesContext: []string{"postId", "postType"}},
		{Name: "core/post-meta", Label: "Post Meta", UsesContext: []string{"postId", "postType"}},
		{Name: "core/term-data", Label: "Term Data", UsesContext: []string{"termId", "taxonomy"}},
	}
	for _, src := range core {
		r.RegisterBlockBindingsSource(src.Name, src)
	}
	if stylesheet == "twentytwentyfive" {
		r.RegisterBlockBindingsSource("twentytwentyfive/format", BlockBindingsSource{
			Name:        "twentytwentyfive/format",
			Label:       "Post format name",
			UsesContext: nil,
		})
	}
}
