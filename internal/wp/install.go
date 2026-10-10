// Install detection stub (P6.4) — is_blog_installed subset via blogname option.
package wp

import "strings"

// IsInstalled reports whether WordPress appears installed.
// Missing / empty blogname (or nil Options) means not installed.
func IsInstalled(c *Context) bool {
	if c == nil || c.Options == nil {
		return false
	}
	v := c.Options.GetOption("blogname", nil)
	if v == nil {
		return false
	}
	s := strings.TrimSpace(optionValueString(v))
	return s != ""
}
