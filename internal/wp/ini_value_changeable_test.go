package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsIniValueChangeable(t *testing.T) {
	wp.ResetIniAccessCacheForTest()
	t.Cleanup(wp.ResetIniAccessCacheForTest)

	if !wp.IsIniValueChangeable("memory_limit") {
		t.Fatal("memory_limit should be changeable (INI_ALL)")
	}
	if wp.IsIniValueChangeable("upload_max_filesize") {
		t.Fatal("upload_max_filesize should not be user-changeable")
	}
	if wp.IsIniValueChangeable("disable_functions") {
		t.Fatal("disable_functions is system-only")
	}
	// Unknown settings fail open (WP when ini_get_all unavailable).
	if !wp.IsIniValueChangeable("totally_unknown_ini_setting_xyz") {
		t.Fatal("unknown setting should fail open to true")
	}
	if !wp.IsIniValueChangeable("") {
		t.Fatal("empty setting should be true")
	}

	wp.SetIniAccessForTest("memory_limit", 4) // INI_SYSTEM
	if wp.IsIniValueChangeable("memory_limit") {
		t.Fatal("overridden memory_limit should be false")
	}
	wp.SetIniAccessForTest("custom_user", 1) // INI_USER
	if !wp.IsIniValueChangeable("custom_user") {
		t.Fatal("INI_USER should be changeable")
	}
}
