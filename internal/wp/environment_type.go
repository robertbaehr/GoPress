// wp_get_environment_type (P2.69) — subset of wp-includes/load.php.
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
// Prefer Config.EnvironmentType (WP_ENVIRONMENT_TYPE define /
// GOPRESS_ENVIRONMENT_TYPE / WP_ENVIRONMENT_TYPE env via config.Load), then
// the process WP_ENVIRONMENT_TYPE variable.
func GetEnvironmentType(c *Context) string {
	env := ""
	if c != nil && c.Config != nil {
		env = strings.TrimSpace(c.Config.EnvironmentType)
	}
	if env == "" {
		if v, ok := os.LookupEnv("WP_ENVIRONMENT_TYPE"); ok {
			env = strings.TrimSpace(v)
		}
	}
	env = strings.ToLower(strings.TrimSpace(env))
	if !allowedEnvironmentTypes[env] {
		return "production"
	}
	return env
}
