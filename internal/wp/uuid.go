// wp_is_uuid (P2.51) — subset of wp-includes/functions.php.
package wp

import (
	"regexp"
	"strings"
)

var (
	reUUIDAny = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	reUUIDV4  = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// IsUUID reports whether uuid matches the UUID pattern (wp_is_uuid).
// When version is 4, only UUID v4 is accepted. Other non-zero versions
// return false (WP only supports V4 and emits _doing_it_wrong).
func IsUUID(uuid string, version int) bool {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return false
	}
	if version != 0 {
		if version != 4 {
			return false
		}
		return reUUIDV4.MatchString(uuid)
	}
	return reUUIDAny.MatchString(uuid)
}
