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

	// ListPosts returns posts for core/query / core/post-template (P5.13).
	ListPosts func(QueryLoopArgs) []QueryPost

	// LoadPattern returns serialized markup for a core/pattern slug.
	LoadPattern func(slug string) (string, bool)

	// LoadFootnotes returns the footnotes post meta JSON for a post ID (P5.15).
	LoadFootnotes func(postID int64) string

	// RenderCommentForm returns WP comment_form HTML for the current post (P5.18).
	RenderCommentForm func(postID int64) string

	// seenPart tracks template-part slugs currently being rendered (cycle guard).
	seenPart map[string]bool

	// seenNav tracks navigation post refs currently being rendered (cycle guard).
	seenNav map[int64]bool

	// currentPost is the post in scope for post-title/content/date (P5.13).
	currentPost *QueryPost

	// queryArgs is set while rendering a core/query tree.
	queryArgs QueryLoopArgs

	// queryHasPosts is set by post-template for query-no-results.
	queryHasPosts bool
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

// SetCurrentPost sets the post in scope for post-* and footnotes blocks (tests/front).
func (e *Env) SetCurrentPost(p *QueryPost) {
	if e == nil {
		return
	}
	e.currentPost = p
}
