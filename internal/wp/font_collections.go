// Font collections library (P4.55) — subset of WP_Font_Library / wp_register_font_collection.
package wp

import (
	"sort"
	"sync"
)

// FontCollectionCategory is a category within a font collection.
type FontCollectionCategory struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// FontCollection is a registered font collection (local metadata only).
// Remote font_families JSON URLs are not fetched (out of scope).
type FontCollection struct {
	Slug         string                   `json:"slug"`
	Name         string                   `json:"name"`
	Description  string                   `json:"description"`
	FontFamilies []any                    `json:"font_families"`
	Categories   []FontCollectionCategory `json:"categories"`
}

// FontLibrary stores registered font collections by slug.
type FontLibrary struct {
	mu          sync.RWMutex
	collections map[string]FontCollection
}

// NewFontLibrary returns an empty font library.
func NewFontLibrary() *FontLibrary {
	return &FontLibrary{collections: map[string]FontCollection{}}
}

// RegisterFontCollection adds or replaces a collection (wp_register_font_collection).
func (l *FontLibrary) RegisterFontCollection(slug string, col FontCollection) bool {
	if l == nil || slug == "" {
		return false
	}
	col.Slug = slug
	if col.FontFamilies == nil {
		col.FontFamilies = []any{}
	}
	if col.Categories == nil {
		col.Categories = []FontCollectionCategory{}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.collections == nil {
		l.collections = map[string]FontCollection{}
	}
	l.collections[slug] = col
	return true
}

// Get returns a collection by slug.
func (l *FontLibrary) Get(slug string) (FontCollection, bool) {
	if l == nil {
		return FontCollection{}, false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	c, ok := l.collections[slug]
	return c, ok
}

// All returns collections sorted by slug.
func (l *FontLibrary) All() []FontCollection {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]FontCollection, 0, len(l.collections))
	for _, c := range l.collections {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

// RegisterDefaultFontCollections registers the google-fonts collection metadata
// without fetching remote font family JSON (P4.55).
func RegisterDefaultFontCollections(l *FontLibrary) {
	if l == nil {
		return
	}
	l.RegisterFontCollection("google-fonts", FontCollection{
		Name:        "Google Fonts",
		Description: "Install from Google Fonts. Fonts are copied to and served from your site.",
		FontFamilies: []any{}, // remote JSON not fetched
		Categories: []FontCollectionCategory{
			{Name: "Sans Serif", Slug: "sans-serif"},
			{Name: "Display", Slug: "display"},
			{Name: "Serif", Slug: "serif"},
			{Name: "Handwriting", Slug: "handwriting"},
			{Name: "Monospace", Slug: "monospace"},
		},
	})
}
