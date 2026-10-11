// hash_equals (P6.48) — timing-safe string compare subset of wp-includes/compat.php / hashing.php.
package wp

import "crypto/subtle"

// HashEquals mirrors WordPress hash_equals(): constant-time compare of two strings.
// Returns false when lengths differ (without leaking which is longer beyond length).
func HashEquals(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
