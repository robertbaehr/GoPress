package blocks

// Env supplies dynamic block resolution during render (P5.6+).
// A nil *Env behaves like an empty env (dynamic blocks render empty).
type Env struct {
	// LoadTemplatePart returns serialized block markup for a template-part slug.
	LoadTemplatePart func(slug string) (string, bool)

	// LoadNavigation returns serialized block markup for a wp_navigation post ID.
	LoadNavigation func(ref int64) (string, bool)

	// ListPages returns fallback navigation links (published pages).
	// Prefer ID/Parent populated for nested core/page-list (P5.75).
	ListPages func() []NavLink

	// CurrentPageID is the queried page ID for current-menu-item (P5.75).
	CurrentPageID int64
	// FrontPageID is page_on_front for menu-item-home (P5.75).
	FrontPageID int64

	// ListPosts returns posts for core/query / core/post-template (P5.13).
	ListPosts func(QueryLoopArgs) []QueryPost

	// LoadPattern returns serialized markup for a core/pattern slug.
	LoadPattern func(slug string) (string, bool)

	// LoadBlock returns serialized markup for a wp_block (synced pattern) post ID (P5.25).
	LoadBlock func(ref int64) (string, bool)

	// LoadFootnotes returns the footnotes post meta JSON for a post ID (P5.15).
	LoadFootnotes func(postID int64) string

	// SanitizeHTML sanitizes footnote (and similar) HTML like wp_kses_post (P5.98).
	SanitizeHTML func(html string) string

	// PasswordRequired reports whether post content (incl. footnotes) is locked (P5.51).
	PasswordRequired func(postID int64) bool

	// RenderCommentForm returns WP comment_form HTML for the current post (P5.18).
	RenderCommentForm func(postID int64) string

	// SiteName / HomeURL / SiteTagline / SiteLogoURL power branding blocks (P5.19/P5.45).
	SiteName    func() string
	SiteTagline func() string
	SiteLogoURL func() string // custom logo / site_logo attachment URL
	// SiteLogoID is the custom logo attachment ID for width/height (P5.120).
	SiteLogoID  func() int64
	HomeURL     func() string
	IsFrontPage func() bool
	// IsHome / IsPaged / QueriedObjectID / PageForPostsID refine home-link + site-title current state (P5.93).
	IsHome          func() bool
	IsPaged         func() bool
	QueriedObjectID func() int64
	PageForPostsID  func() int64

	// IsLoggedIn / CurrentURL / LoginURL / LogoutURL power core/loginout (P5.74).
	IsLoggedIn func() bool
	CurrentURL func() string
	LoginURL   func(redirect string) string
	LogoutURL  func(redirect string) string

	// GetAttachmentURL resolves attachment IDs for core/image and core/gallery (P5.48).
	GetAttachmentURL func(id int64) string
	// GetAttachmentDimensions returns width/height from attachment metadata (P5.114 / core/video).
	GetAttachmentDimensions func(id int64) (width, height int)
	// AdjacentPost returns prev/next post for post-navigation-link (type previous|next).
	// taxonomy is the in_same_term taxonomy slug when set (empty = unrestricted).
	AdjacentPost func(postID int64, previous bool, taxonomy string) *QueryPost

	// FilterPostHTML runs after rendering post_content blocks (texturize etc.).
	FilterPostHTML func(html string) string

	// ListComments returns approved comments for the current post (P5.19).
	ListComments func(postID int64) []CommentItem

	// CommentOrder is comment_order option (asc|desc) for core/comment-template (P5.90).
	CommentOrder string

	// PageComments enables front comment paging (page_comments option, P3.108).
	PageComments bool
	// CommentsPerPage is comments_per_page when PageComments is on (P3.108).
	CommentsPerPage int
	// CommentPage is the resolved 1-based cpage (P3.108).
	CommentPage int
	// CommentTotalPages is total comment pages (≥1 when paging, P3.108).
	CommentTotalPages int
	// CommentPermalink is the singular URL used for cpage links (P3.108).
	CommentPermalink string

	// AuthorName / AuthorEmail / AuthorBio / AuthorLink power author blocks (P5.19/P5.46).
	AuthorName  func(postID int64) string
	AuthorEmail func(postID int64) string
	AuthorBio   func(postID int64) string
	AuthorLink  func(postID int64) string
	PostTerms   func(postID int64, taxonomy string) []NavLink

	// GetPostMeta resolves core/post-meta block bindings (P5.20).
	GetPostMeta func(postID int64, key string) string

	// BindingAttrsFilter mirrors block_bindings_supported_attributes{,_$type} (P9.25).
	// Receives block type + default supported attrs; return the effective list.
	BindingAttrsFilter func(blockType string, attrs []string) []string

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

	// ThemeSupports mirrors current_theme_supports for block opt-ins (P5.124 search-element).
	// feature is e.g. "html5"; args are optional sub-features like "search-element".
	ThemeSupports func(feature string, args ...any) bool

	// ArchiveTitle is the front date/term archive heading for core/query-title type=archive (P3.22).
	ArchiveTitle string

	// ListArchives returns archive links for core/archives (P5.42 / P5.122).
	// archiveType is monthly|yearly|daily|weekly (default monthly).
	ListArchives func(archiveType string) []ArchiveItem

	// HasPublishedPosts reports whether any published post exists (core/calendar, P5.122).
	// When false, logged-in users see a notice; guests get empty markup.
	HasPublishedPosts func() bool

	// ListTagCloud returns terms for core/tag-cloud (P5.42 / P5.82).
	// taxonomy defaults to post_tag; number caps the result (WP numberOfTags, default 45).
	ListTagCloud func(taxonomy string, number int) []TagCloudItem

	// ListCategories returns terms for core/categories (P5.81).
	ListCategories func(taxonomy string) []CategoryItem

	// ListLatestComments returns recent approved comments for core/latest-comments (P5.42).
	ListLatestComments func(limit int) []LatestCommentItem

	// seenPart tracks template-part slugs currently being rendered (cycle guard).
	seenPart map[string]bool

	// seenNav tracks navigation post refs currently being rendered (cycle guard).
	seenNav map[int64]bool

	// renderingNavigation is true while rendering a core/navigation tree (P5.75).
	renderingNavigation bool
	// pageListNav carries Navigation attrs for page-list children (P5.94).
	pageListNav pageListNavCtx

	// seenBlock tracks wp_block refs currently being rendered (cycle guard, P5.25).
	seenBlock map[int64]bool

	// seenPostContent tracks post IDs while rendering core/post-content (P5.87).
	// Mirrors PHP $seen_ids reentrancy guard in render_block_core_post_content.
	seenPostContent map[int64]bool

	// DebugDisplay, when true, surfaces post-content reentrancy warnings (P5.87).
	// Mirrors WP_DEBUG && WP_DEBUG_DISPLAY in PHP.
	DebugDisplay bool

	// currentPost is the post in scope for post-title/content/date (P5.13).
	currentPost *QueryPost

	// queryArgs is set while rendering a core/query tree.
	queryArgs QueryLoopArgs

	// queryHasPosts is set by post-template for query-no-results.
	queryHasPosts bool

	// Paged is the current main-query page (1-based) for query-pagination (P5.44).
	Paged int
	// MaxPages is max_num_pages for inherit query pagination (P5.44).
	MaxPages int

	// paginationArrow / showLabel are context from enclosing core/query-pagination (P5.119).
	// Empty paginationArrow means "arrow" (legacy default when unset).
	paginationArrow string
	showLabel       *bool // nil → true

	// socialCtx is context from an enclosing core/social-links block (P5.50).
	socialCtx socialLinksCtx
}

// NavLink is a simple navigation item (label + URL).
// ID/Parent enable nested page-list rendering (P5.75).
type NavLink struct {
	Label  string
	URL    string
	ID     int64
	Parent int64
}

// CommentItem is a comment row for core/comment-template (P5.19 / P5.90 / P3.116).
type CommentItem struct {
	ID           int64
	Parent       int64
	Author       string
	AuthorEmail  string
	AuthorURL    string
	Content      string
	Date         string
	UserID       int64  // registered commenter (P3.116 get_comment_class)
	UserNicename string // for comment-author-{nicename}
	PostAuthor   int64  // post author for bypostauthor
	Type         string // comment|trackback|pingback
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
