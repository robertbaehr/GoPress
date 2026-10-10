// wp_is_ini_value_changeable (P2.63) — subset of wp-includes/load.php.
package wp

import "sync"

// PHP ini access level constants (php.ini / ini_get_all).
const (
	iniUser   = 1 // INI_USER
	iniPerDir = 2 // INI_PERDIR
	iniSystem = 4 // INI_SYSTEM
	iniAll    = 7 // INI_ALL
)

var (
	iniAccessMu    sync.Mutex
	iniAccessCache map[string]int // setting → access bitmask; nil until primed
)

// Typical PHP access levels for settings WordPress probes (memory_limit, etc.).
// When a setting is absent, IsIniValueChangeable fails open (true), matching WP
// when ini_get_all() is disabled via disable_functions.
var defaultIniAccess = map[string]int{
	"memory_limit":           iniAll,
	"max_execution_time":     iniAll,
	"display_errors":         iniAll,
	"log_errors":             iniAll,
	"error_reporting":        iniAll,
	"upload_max_filesize":    iniSystem | iniPerDir, // not user-changeable
	"post_max_size":          iniSystem | iniPerDir,
	"disable_functions":      iniSystem,
	"open_basedir":           iniSystem,
	"allow_url_fopen":        iniSystem,
	"zend.assertions":        iniAll,
	"precision":              iniAll,
	"serialize_precision":    iniAll,
	"default_socket_timeout": iniAll,
}

// IsIniValueChangeable reports whether setting can be changed at runtime
// (wp_is_ini_value_changeable). True when access is INI_ALL or INI_USER, or when
// the setting is unknown (fail open).
func IsIniValueChangeable(setting string) bool {
	if setting == "" {
		return true
	}
	access, ok := iniAccessLevel(setting)
	if !ok {
		return true
	}
	return access == iniAll || access == iniUser
}

func iniAccessLevel(setting string) (int, bool) {
	iniAccessMu.Lock()
	defer iniAccessMu.Unlock()
	if iniAccessCache == nil {
		iniAccessCache = make(map[string]int, len(defaultIniAccess))
		for k, v := range defaultIniAccess {
			iniAccessCache[k] = v
		}
	}
	v, ok := iniAccessCache[setting]
	return v, ok
}

// ResetIniAccessCacheForTest clears the memoized access table (tests only).
func ResetIniAccessCacheForTest() {
	iniAccessMu.Lock()
	iniAccessCache = nil
	iniAccessMu.Unlock()
}

// SetIniAccessForTest overrides a setting's access level (tests only).
func SetIniAccessForTest(setting string, access int) {
	iniAccessMu.Lock()
	defer iniAccessMu.Unlock()
	if iniAccessCache == nil {
		iniAccessCache = make(map[string]int, len(defaultIniAccess))
		for k, v := range defaultIniAccess {
			iniAccessCache[k] = v
		}
	}
	iniAccessCache[setting] = access
}
