// Block pattern categories registry (P5.38) — subset of
// WP_Block_Pattern_Categories_Registry / register_block_pattern_category.
package wp

import (
	"sort"
	"sync"
)

// PatternCategory is a registered block pattern category.
type PatternCategory struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// PatternCategoryRegistry stores registered pattern categories by name.
type PatternCategoryRegistry struct {
	mu         sync.RWMutex
	categories map[string]PatternCategory
}

// NewPatternCategoryRegistry returns an empty registry.
func NewPatternCategoryRegistry() *PatternCategoryRegistry {
	return &PatternCategoryRegistry{categories: map[string]PatternCategory{}}
}

// RegisterPatternCategory adds or replaces a category (register_block_pattern_category).
func (r *PatternCategoryRegistry) RegisterPatternCategory(name string, cat PatternCategory) bool {
	if r == nil || name == "" {
		return false
	}
	if cat.Label == "" {
		cat.Label = name
	}
	cat.Name = name
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.categories == nil {
		r.categories = map[string]PatternCategory{}
	}
	r.categories[name] = cat
	return true
}

// UnregisterPatternCategory removes a category by name.
func (r *PatternCategoryRegistry) UnregisterPatternCategory(name string) bool {
	if r == nil || name == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.categories[name]; !ok {
		return false
	}
	delete(r.categories, name)
	return true
}

// Get returns a category by name.
func (r *PatternCategoryRegistry) Get(name string) (PatternCategory, bool) {
	if r == nil {
		return PatternCategory{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.categories[name]
	return c, ok
}

// All returns registered categories sorted by name for stable REST output.
func (r *PatternCategoryRegistry) All() []PatternCategory {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]PatternCategory, 0, len(r.categories))
	for _, c := range r.categories {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// RegisterDefaultPatternCategories registers core categories from
// wp-includes/block-patterns.php::_register_core_block_patterns_and_categories
// plus active-theme categories when known (Twenty Twenty-Five).
func RegisterDefaultPatternCategories(r *PatternCategoryRegistry, stylesheet string) {
	if r == nil {
		return
	}
	core := []PatternCategory{
		{Name: "banner", Label: "Banners", Description: "Bold sections designed to showcase key content."},
		{Name: "buttons", Label: "Buttons", Description: "Patterns that contain buttons and call to actions."},
		{Name: "columns", Label: "Columns", Description: "Multi-column patterns with more complex layouts."},
		{Name: "text", Label: "Text", Description: "Patterns containing mostly text."},
		{Name: "query", Label: "Posts", Description: "Display your latest posts in lists, grids or other layouts."},
		{Name: "featured", Label: "Featured", Description: "A set of high quality curated patterns."},
		{Name: "call-to-action", Label: "Call to action", Description: "Sections whose purpose is to trigger a specific action."},
		{Name: "team", Label: "Team", Description: "A variety of designs to display your team members."},
		{Name: "testimonials", Label: "Testimonials", Description: "Share reviews and feedback about your brand/business."},
		{Name: "services", Label: "Services", Description: "Briefly describe what your business does and how you can help."},
		{Name: "contact", Label: "Contact", Description: "Display your contact information."},
		{Name: "about", Label: "About", Description: "Introduce yourself."},
		{Name: "portfolio", Label: "Portfolio", Description: "Showcase your latest work."},
		{Name: "gallery", Label: "Gallery", Description: "Different layouts for displaying images."},
		{Name: "media", Label: "Media", Description: "Different layouts containing video or audio."},
		{Name: "videos", Label: "Videos", Description: "Different layouts containing videos."},
		{Name: "audio", Label: "Audio", Description: "Different layouts containing audio."},
		{Name: "posts", Label: "Posts", Description: "Display your latest posts in lists, grids or other layouts."},
		{Name: "footer", Label: "Footers", Description: "A variety of footer designs displaying information and site navigation."},
		{Name: "header", Label: "Headers", Description: "A variety of header designs displaying your site title and navigation."},
		{Name: "navigation", Label: "Navigation", Description: "A variety of designs displaying site navigation."},
	}
	for _, cat := range core {
		r.RegisterPatternCategory(cat.Name, cat)
	}
	if stylesheet == "twentytwentyfive" {
		r.RegisterPatternCategory("twentytwentyfive_page", PatternCategory{
			Label:       "Pages",
			Description: "A collection of full page layouts.",
		})
		r.RegisterPatternCategory("twentytwentyfive_post-format", PatternCategory{
			Label:       "Post formats",
			Description: "A collection of post format patterns.",
		})
	}
}
