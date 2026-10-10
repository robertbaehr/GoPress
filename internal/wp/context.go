// Package wp is the WordPress runtime core (hooks, options, query, …).
//
// Rule 0.5 / P1.1: request state lives on Context and RequestScope — never in
// package-level mutable globals.
package wp

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/db"
	"github.com/robertbaehr/GoPress/internal/theme"
)

// ctxKey is the private context.Context key for *Context.
type ctxKey struct{}

// Context is the per-request (or per-CLI-invocation) WordPress runtime.
// Later P1 tasks attach concrete implementations; until then fields may be nil.
type Context struct {
	// Parent is the standard library context for cancellation/deadlines.
	Parent context.Context

	// Config is site configuration (P1.2).
	Config *config.Config

	// DB is the database handle (P1.3).
	DB *db.DB

	// Hooks is the action/filter registry for this process (P1.4).
	// Typically one process-wide *Hooks shared across requests.
	Hooks *Hooks

	// Cache is the object cache (P1.6).
	Cache *ObjectCache

	// Options is the options API (P1.7).
	Options *Options

	// Meta is the metadata API (P1.8).
	Meta *Meta

	// Locale is the active locale / translations (P1.9).
	Locale *Locale

	// Auth provides salts, hashes, nonces, and auth cookies (P2.1+).
	Auth *Auth

	// Users is the user lookup API (P2.2).
	Users *Users

	// Roles holds wp_user_roles and UserCan helpers (P2.3).
	Roles *Roles

	// Theme is the active theme.json settings/styles subset (P5.4).
	Theme *theme.JSON

	// Patterns is the block pattern registry (P5.9).
	Patterns *PatternRegistry

	// PatternCategories is the block pattern category registry (P5.38).
	PatternCategories *PatternCategoryRegistry

	// BlockStyles is the block styles registry (P5.26).
	BlockStyles *BlockStyleRegistry

	// BlockVariations is the block variations registry (P5.37).
	BlockVariations *BlockVariationRegistry

	// BlockBindings is the block bindings sources registry (P5.39).
	BlockBindings *BlockBindingsRegistry

	// FontLibrary holds registered font collections (P4.55).
	FontLibrary *FontLibrary

	// Shortcodes is the shortcode tag registry (P2.11).
	Shortcodes *Shortcodes

	// Mailer records wp_mail sends (P3.9).
	Mailer *Mailer

	// CurrentUser is the authenticated user for this request (P2.2+).
	CurrentUser *User

	// SessionToken is the session token from the auth cookie (P2.4+/P7.2).
	SessionToken string

	// Req is the inbound HTTP request (nil for CLI).
	Req *http.Request

	// Res is the HTTP response writer (nil for CLI).
	Res http.ResponseWriter

	// Scope holds lazily computed request-local WordPress state.
	Scope *RequestScope
}

// NewContext builds a Context with an empty RequestScope.
func NewContext(parent context.Context) *Context {
	if parent == nil {
		parent = context.Background()
	}
	return &Context{
		Parent: parent,
		Scope:  NewRequestScope(),
	}
}

// ForRequest returns a per-request Context sharing process-wide services
// (Config, DB, Hooks, Cache, Options, …) with a fresh Scope and no CurrentUser.
func (c *Context) ForRequest(w http.ResponseWriter, r *http.Request) *Context {
	if c == nil {
		out := NewContext(context.Background())
		out.Req = r
		out.Res = w
		return out
	}
	parent := c.Parent
	if r != nil {
		parent = r.Context()
	}
	return &Context{
		Parent:            parent,
		Config:            c.Config,
		DB:                c.DB,
		Hooks:             c.Hooks,
		Cache:             c.Cache,
		Options:           c.Options,
		Meta:              c.Meta,
		Locale:            c.Locale,
		Auth:              c.Auth,
		Users:             c.Users,
		Roles:             c.Roles,
		Theme:             c.Theme,
		Patterns:          c.Patterns,
		PatternCategories: c.PatternCategories,
		BlockStyles:       c.BlockStyles,
		BlockVariations:   c.BlockVariations,
		BlockBindings:     c.BlockBindings,
		FontLibrary:       c.FontLibrary,
		Shortcodes:        c.Shortcodes,
		Mailer:            c.Mailer,
		Req:               r,
		Res:               w,
		Scope:             NewRequestScope(),
	}
}

// WithContext stores c in parent and returns the derived context.Context.
func WithContext(parent context.Context, c *Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	c.Parent = parent
	return context.WithValue(parent, ctxKey{}, c)
}

// FromContext returns the *Context stored in ctx, or nil.
func FromContext(ctx context.Context) *Context {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(ctxKey{}).(*Context)
	return c
}

// RequestScope holds per-request WordPress state computed at most once.
// Mirrors the role of globals like $wp_query / $post without package mutables.
type RequestScope struct {
	mu sync.Mutex

	queryParsed sync.Once
	query       any
	queryErr    error

	postOnce sync.Once
	post     any
	postErr  error

	// QueriedObjectID is set by the rewrite/query layer (P3.x).
	QueriedObjectID int64

	// SearchQuery is the `s` query var when IsSearch (P3.21).
	SearchQuery string

	// Date archive query vars (P3.22).
	DateYear  int
	DateMonth int
	DateDay   int
	IsDate    bool
	IsYear    bool
	IsMonth   bool
	IsDay     bool

	// Author archive query vars (P3.23).
	AuthorID   int64
	AuthorName string
	IsAuthor   bool

	// Tag archive query vars (P3.24).
	TagSlug string
	TagID   int64
	IsTag   bool

	// Category archive query vars (P3.25).
	CategorySlug string
	CategoryID   int64
	IsCategory   bool

	// Is* flags mirror WP_Query conditionals (filled by query parse).
	IsHome       bool
	IsSingular   bool
	IsArchive    bool
	IsSearch     bool
	Is404        bool
	IsAdmin      bool
	IsREST       bool
	IsFeed       bool
	IsFrontPage  bool
	IsPostsPage  bool
	IsPage       bool
	IsSingle     bool
	IsAttachment bool

	// Paged is the main query page number (1-based; 0 means unset → page 1).
	Paged int
}

// NewRequestScope returns an empty scope.
func NewRequestScope() *RequestScope {
	return &RequestScope{}
}

// DoQueryParse runs fn at most once for this scope and caches the result.
func (s *RequestScope) DoQueryParse(fn func() (any, error)) (any, error) {
	if s == nil {
		return nil, errNilScope
	}
	s.queryParsed.Do(func() {
		s.query, s.queryErr = fn()
	})
	return s.query, s.queryErr
}

// Query returns the cached query result (nil before DoQueryParse).
func (s *RequestScope) Query() any {
	if s == nil {
		return nil
	}
	return s.query
}

// DoCurrentPost runs fn at most once and caches the current post.
func (s *RequestScope) DoCurrentPost(fn func() (any, error)) (any, error) {
	if s == nil {
		return nil, errNilScope
	}
	s.postOnce.Do(func() {
		s.post, s.postErr = fn()
	})
	return s.post, s.postErr
}

// CurrentPost returns the cached post (nil before DoCurrentPost).
func (s *RequestScope) CurrentPost() any {
	if s == nil {
		return nil
	}
	return s.post
}

// SetFlag updates a named conditional under the scope lock.
// Reserved for query bootstrap; prefer typed fields when possible.
func (s *RequestScope) SetAdmin(v bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.IsAdmin = v
	s.mu.Unlock()
}

// SetREST marks the request as a REST API request.
func (s *RequestScope) SetREST(v bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.IsREST = v
	s.mu.Unlock()
}

type scopeError string

func (e scopeError) Error() string { return string(e) }

const errNilScope = scopeError("wp: nil RequestScope")

// AdminNotice appends HTML emitted during admin_notices (into Hooks capture).
func (c *Context) AdminNotice(html string) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.WriteNotice(html)
}

// CaptureAdminNotices runs the admin_notices action and returns buffered HTML.
func (c *Context) CaptureAdminNotices() string {
	if c == nil || c.Hooks == nil {
		return ""
	}
	var buf strings.Builder
	c.Hooks.BeginNoticeCapture(&buf)
	defer c.Hooks.EndNoticeCapture()
	c.Hooks.DoAction("admin_notices")
	return buf.String()
}
