package wp

import (
	"net/http"
	"strings"
	"sync"

	"github.com/robertbaehr/GoPress/internal/config"
)

var (
	forceSSLAdminMu sync.Mutex
	forceSSLAdmin   bool
)

// ForceSSLAdmin gets or sets the FORCE_SSL_ADMIN flag (force_ssl_admin, P6.40).
// When force is non-nil, sets the static flag and returns the previous value
// (PHP force_ssl_admin( $force ) semantics). When force is nil, returns the
// current flag.
func ForceSSLAdmin(force *bool) bool {
	forceSSLAdminMu.Lock()
	defer forceSSLAdminMu.Unlock()
	if force != nil {
		prev := forceSSLAdmin
		forceSSLAdmin = *force
		return prev
	}
	return forceSSLAdmin
}

// ApplyForceSSLAdminFromConfig initializes force_ssl_admin from FORCE_SSL_ADMIN
// (default-constants.php + force_ssl_admin( FORCE_SSL_ADMIN )).
func ApplyForceSSLAdminFromConfig(cfg *config.Config) {
	if cfg == nil {
		return
	}
	v, ok := cfg.Extra["FORCE_SSL_ADMIN"]
	if !ok {
		return
	}
	on := phpTruthy(v)
	ForceSSLAdmin(&on)
}

func phpTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// MaybeRedirectLoginToHTTPS redirects wp-login.php to HTTPS when
// force_ssl_admin() && ! is_ssl() (wp-login.php top, P6.40).
// Returns true when a redirect was written.
func MaybeRedirectLoginToHTTPS(w http.ResponseWriter, r *http.Request, c *Context) bool {
	if !ForceSSLAdmin(nil) || IsSSL(c) {
		return false
	}
	if r == nil || w == nil {
		return false
	}
	target := ""
	reqURI := ""
	if r.URL != nil {
		reqURI = r.URL.RequestURI()
	}
	if reqURI == "" {
		reqURI = "/wp-login.php"
	}
	if strings.HasPrefix(strings.ToLower(reqURI), "http://") || strings.HasPrefix(strings.ToLower(reqURI), "https://") {
		target = SetURLScheme(c, reqURI, "https")
	} else {
		host := r.Host
		if host == "" && r.URL != nil {
			host = r.URL.Host
		}
		if host == "" {
			host = "localhost"
		}
		target = "https://" + host + reqURI
	}
	if target == "" {
		return false
	}
	// Prefer ValidateRedirect (wp_safe_redirect); fall back to the constructed
	// HTTPS URL when home host is unset in tests / early boot.
	location := ValidateRedirect(c, target, target)
	if location == "" {
		location = target
	}
	http.Redirect(w, r, location, http.StatusFound)
	return true
}
