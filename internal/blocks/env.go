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

	// LoadBlock returns serialized markup for a wp_block (synced pattern) post ID (P5.25).
	LoadBlock func(ref int64) (string, bool)

	// LoadFootnotes returns the footnotes post meta JSON for a post ID (P5.15).
	LoadFootnotes func(postID int64) string

	// RenderCommentForm returns WP comment_form HTML for the current post (P5.18).
	RenderCommentForm func(postID int64) string

	// SiteName / HomeURL power core/site-title and related blocks (P5.19).
	SiteName    func() string
	HomeURL     func() string
	IsFrontPage func() bool
	// AdjacentPost returns prev/next post for post-navigation-link (type previous|next).
	AdjacentPost func(postID int64, previous bool) *QueryPost

	// FilterPostHTML runs after rendering post_content blocks (texturize etc.).
	FilterPostHTML func(html string) string

	// ListComments returns approved comments for the current post (P5.19).
	ListComments func(postID int64) []CommentItem

	// AuthorName / PostTerms power singular meta blocks (P5.19).
	AuthorName func(postID int64) string
	PostTerms  func(postID int64, taxonomy string) []NavLink

	// GetPostMeta resolves core/post-meta block bindings (P5.20).
	GetPostMeta func(postID int64, key string) string

	// PatternOverrides is context for core/pattern-overrides bindings (P5.20/P5.25).
	// Keys are metadata.name values; values map attribute → override string.
	// Populated from core/block content|overrides attrs (providesContext pattern/overrides).
	PatternOverrides map[string]map[string]string

	// TermContext powers core/term-data block bindings (P5.39).
	TermContext *TermBindingContext

	// PostFormat returns the current post format slug for theme bindings (P5.39).
	PostFormat func(postID int64) string

	// SearchQuery is the front `s` term for core/query-title type=search (P3.21).
	SearchQuery string

	// ArchiveTitle is the front date/term archive heading for core/query-title type=archive (P3.22).
	ArchiveTitle string

	// ListArchives returns monthly archive links for core/archives (P5.42).
	ListArchives func() []ArchiveItem

	// ListTagCloud returns tags for core/tag-cloud (P5.42).
	ListTagCloud func() []TagCloudItem

	// ListLatestComments returns recent approved comments for core/latest-comments (P5.42).
	ListLatestComments func(limit int) []LatestCommentItem

	// seenPart tracks template-part slugs currently being rendered (cycle guard).
	seenPart map[string]bool

	// seenNav tracks navigation post refs currently being rendered (cycle guard).
	seenNav map[int64]bool

	// seenBlock tracks wp_block refs currently being rendered (cycle guard, P5.25).
	seenBlock map[int64]bool

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

// CommentItem is a comment row for core/comment-template (P5.19).
type CommentItem struct {
	ID          int64
	Author      string
	AuthorEmail string
	AuthorURL   string
	Content     string
	Date        string
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
