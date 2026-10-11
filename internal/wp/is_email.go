// is_email (P2.106) — subset of wp-includes/formatting.php.
package wp

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reEmailLocal     = regexp.MustCompile(`^[a-zA-Z0-9!#$%&'*+/=?^_` + "`" + `{|}~.-]+$`)
	reEmailDomainSub = regexp.MustCompile(`(?i)^[a-z0-9-]+$`)
	reEmailDots      = regexp.MustCompile(`\.{2,}`)
)

// IsEmail validates an email address (is_email without the is_email filter).
// Returns the email when valid, or "" when invalid — matching callers that
// treat a non-empty return as success (WP returns the address or false).
func IsEmail(email string) string {
	ok, _ := checkIsEmail(email)
	return ok
}

// IsEmailValid reports whether email passes WordPress is_email checks.
func IsEmailValid(email string) bool {
	return IsEmail(email) != ""
}

// IsEmailFiltered mirrors is_email() including the is_email filter (P2.106).
// On success the filter receives (email, email, nil); on failure
// (false, email, context) where context is one of the WP failure codes.
func IsEmailFiltered(c *Context, email string) string {
	ok, ctx := checkIsEmail(email)
	if c == nil || c.Hooks == nil {
		return ok
	}
	var value any
	if ok != "" {
		value = ok
	} else {
		value = false
	}
	var filterCtx any
	if ctx != "" {
		filterCtx = ctx
	}
	filtered := c.Hooks.ApplyFilters("is_email", value, email, filterCtx)
	switch t := filtered.(type) {
	case string:
		return t
	case bool:
		if t {
			return email
		}
		return ""
	default:
		if filtered == nil {
			return ""
		}
		s := strings.TrimSpace(fmt.Sprint(filtered))
		if s == "" || s == "false" || s == "0" {
			return ""
		}
		if s == "true" || s == "1" {
			return email
		}
		return s
	}
}

func checkIsEmail(email string) (string, string) {
	// WP does not trim; keep exact input for successful return.
	if len(email) < 6 {
		return "", "email_too_short"
	}
	// Test for an @ character after the first position.
	if !strings.Contains(email[1:], "@") {
		return "", "email_no_at"
	}
	local, domain, ok := strings.Cut(email, "@")
	if !ok {
		return "", "email_no_at"
	}
	if !reEmailLocal.MatchString(local) {
		return "", "local_invalid_chars"
	}
	if reEmailDots.MatchString(domain) {
		return "", "domain_period_sequence"
	}
	if strings.Trim(domain, " \t\n\r\x00\x0B.") != domain {
		return "", "domain_period_limits"
	}
	subs := strings.Split(domain, ".")
	if len(subs) < 2 {
		return "", "domain_no_periods"
	}
	for _, sub := range subs {
		if strings.Trim(sub, " \t\n\r\x00\x0B-") != sub {
			return "", "sub_hyphen_limits"
		}
		if !reEmailDomainSub.MatchString(sub) {
			return "", "sub_invalid_chars"
		}
	}
	return email, ""
}
