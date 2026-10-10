package blocks

// Env supplies dynamic block resolution during render (P5.6+).
// A nil *Env behaves like an empty env (dynamic blocks render empty).
type Env struct {
	// LoadTemplatePart returns serialized block markup for a template-part slug.
	LoadTemplatePart func(slug string) (string, bool)

	// LoadNavigation returns serialized block markup for a wp_navigation post ID.
	LoadNavigation func(ref int64) (string, bool)

	// ListPages returns fallback navigation links (published pages).
	ListPages func() []NavLink

	// seenPart tracks template-part slugs currently being rendered (cycle guard).
	seenPart map[string]bool
}

// NavLink is a simple navigation item (label + URL).
type NavLink struct {
	Label string
	URL   string
}

func (e *Env) ensure() *Env {
	if e == nil {
		return &Env{}
	}
	return e
}
