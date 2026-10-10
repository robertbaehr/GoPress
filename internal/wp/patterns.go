// Block pattern registry (P5.9) — theme files + wp_block posts.
package wp

import (
	"fmt"
	"sync"

	"github.com/robertbaehr/GoPress/internal/theme"
)

// BlockPattern is a registered block pattern (subset of WP_Block_Patterns_Registry).
type BlockPattern struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Content     string   `json:"content"`
	Categories  []string `json:"categories,omitempty"`
	BlockTypes  []string `json:"blockTypes,omitempty"`
	Inserter    bool     `json:"inserter"`
	Source      string   `json:"source,omitempty"` // theme | wp_block
}

// PatternRegistry stores registered block patterns.
type PatternRegistry struct {
	mu       sync.RWMutex
	patterns map[string]BlockPattern
	loaded   bool
}

// NewPatternRegistry returns an empty registry.
func NewPatternRegistry() *PatternRegistry {
	return &PatternRegistry{patterns: map[string]BlockPattern{}}
}

// RegisterBlockPattern adds or replaces a pattern by name.
func (r *PatternRegistry) RegisterBlockPattern(name string, p BlockPattern) {
	if r == nil || name == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.patterns == nil {
		r.patterns = map[string]BlockPattern{}
	}
	p.Name = name
	r.patterns[name] = p
}

// Get returns a pattern by name.
func (r *PatternRegistry) Get(name string) (BlockPattern, bool) {
	if r == nil {
		return BlockPattern{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.patterns[name]
	return p, ok
}

// All returns registered patterns in map iteration order (stable enough for stubs).
func (r *PatternRegistry) All() []BlockPattern {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]BlockPattern, 0, len(r.patterns))
	for _, p := range r.patterns {
		out = append(out, p)
	}
	return out
}

// EnsurePatterns loads theme + DB patterns once onto c.Patterns.
func EnsurePatterns(c *Context) error {
	if c == nil {
		return fmt.Errorf("wp: EnsurePatterns: nil context")
	}
	if c.Patterns == nil {
		c.Patterns = NewPatternRegistry()
	}
	c.Patterns.mu.Lock()
	already := c.Patterns.loaded
	c.Patterns.mu.Unlock()
	if already {
		return nil
	}
	return LoadPatterns(c)
}

// LoadPatterns registers patterns from the active theme and published wp_block posts.
func LoadPatterns(c *Context) error {
	if c == nil {
		return fmt.Errorf("wp: LoadPatterns: nil context")
	}
	if c.Patterns == nil {
		c.Patterns = NewPatternRegistry()
	}
	stylesheet := ActiveStylesheet(c)
	abs := AbsPath(c)
	files, err := theme.LoadThemePatterns(abs, stylesheet)
	if err != nil {
		return err
	}
	for _, f := range files {
		c.Patterns.RegisterBlockPattern(f.Slug, BlockPattern{
			Name:        f.Slug,
			Title:       f.Title,
			Description: f.Description,
			Content:     f.Content,
			Categories:  f.Categories,
			BlockTypes:  f.BlockTypes,
			Inserter:    f.Inserter,
			Source:      "theme",
		})
	}
	if err := loadDBBlockPatterns(c); err != nil {
		return err
	}
	c.Patterns.mu.Lock()
	c.Patterns.loaded = true
	c.Patterns.mu.Unlock()
	return nil
}

func loadDBBlockPatterns(c *Context) error {
	if c == nil || c.DB == nil {
		return nil
	}
	rows, err := GetPosts(c, QueryArgs{
		PostType:     "wp_block",
		PostStatus:   "publish",
		PostsPerPage: -1,
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		name := row.Name
		if name == "" {
			name = fmt.Sprintf("core/block/%d", row.ID)
		} else if !containsSlash(name) {
			name = "core/block/" + name
		}
		title := row.Title
		if title == "" {
			title = name
		}
		c.Patterns.RegisterBlockPattern(name, BlockPattern{
			Name:     name,
			Title:    title,
			Content:  row.Content,
			Inserter: true,
			Source:   "wp_block",
		})
	}
	return nil
}

func containsSlash(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return true
		}
	}
	return false
}
