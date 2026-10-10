// wp_get_environment_type (P2.69 / P2.82) — subset of wp-includes/load.php.
package wp

import (
	"os"
	"strings"
)

var allowedEnvironmentTypes = map[string]bool{
	"local":       true,
	"development": true,
	"staging":     true,
	"production":  true,
}

// GetEnvironmentType returns the current environment type
// (wp_get_environment_type). Allowed values: local, development, staging,
// production. Invalid or empty values fall back to production.
//
// Resolution matches WordPress load.php order (P2.82):
//  1. process WP_ENVIRONMENT_TYPE (getenv), when present
//  2. Config.EnvironmentType (WP_ENVIRONMENT_TYPE define /
//     GOPRESS_ENVIRONMENT_TYPE via config.Load) overrides getenv
//
// Values are trimmed and lowercased before the allow-list check. The deprecated
// WP_ENVIRONMENT_TYPES constant is ignored (WordPress 5.5.1+).
func GetEnvironmentType(c *Context) string {
	env := ""
	// Step 1: getenv — presence matters (including empty → invalid → production).
	if v, ok := os.LookupEnv("WP_ENVIRONMENT_TYPE"); ok {
		env = v
	}
	// Step 2: define / config overrides the process environment.
	if c != nil && c.Config != nil {
		if cfg := strings.TrimSpace(c.Config.EnvironmentType); cfg != "" {
			env = cfg
		}
	}
	env = strings.ToLower(strings.TrimSpace(env))
	if !allowedEnvironmentTypes[env] {
		return "production"
	}
	return env
}

// IsEnvironmentType reports whether GetEnvironmentType equals typ
// (case-insensitive). Unknown typ values never match.
func IsEnvironmentType(c *Context, typ string) bool {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if !allowedEnvironmentTypes[typ] {
		return false
	}
	return GetEnvironmentType(c) == typ
}
