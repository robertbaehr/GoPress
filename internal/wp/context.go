// Package wp is the WordPress runtime core (hooks, options, query, …).
//
// Rule 0.5 / P1.1: request state lives on Context and RequestScope — never in
// package-level mutable globals.
package wp

import (
	"context"
	"net/http"
	"sync"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/db"
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

	// CurrentUser is the authenticated user for this request (P2.2+).
	CurrentUser *User

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
