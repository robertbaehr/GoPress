// Custom logo helpers (P3.111) — get_custom_logo / has_custom_logo / the_custom_logo subset.
package wp

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

// CustomLogoID returns the custom logo attachment ID from theme_mod `custom_logo`,
// falling back to the `site_logo` option (block themes).
func CustomLogoID(c *Context) int64 {
	if c == nil {
		return 0
	}
	mods := loadThemeMods(c)
	if id := anyToInt64(mods["custom_logo"]); id > 0 {
		return id
	}
	if c.Options == nil {
		return 0
	}
	return anyToInt64(c.Options.GetOption("site_logo", 0))
}

// AttachmentIsImage reports whether an attachment is an image (wp_attachment_is_image, P8.51).
// Delegates to AttachmentIs("image", …); falls back to resolvable image src when mime/file missing.
func AttachmentIsImage(c *Context, attachmentID int64) bool {
	if AttachmentIs(c, "image", attachmentID) {
		return true
	}
	if c == nil || attachmentID <= 0 {
		return false
	}
	// Fallback: resolvable image src when attached file / mime unavailable.
	_, ok := GetAttachmentImageSrc(c, attachmentID, "full", false)
	return ok
}

// HasCustomLogo reports whether the site has a custom logo image (has_custom_logo).
func HasCustomLogo(c *Context) bool {
	id := CustomLogoID(c)
	if id <= 0 {
		return false
	}
	return AttachmentIsImage(c, id)
}

// GetCustomLogo returns custom logo HTML linked to home (get_custom_logo subset, P3.111).
// Honors unlink-homepage-logo theme support, attachment alt meta, and
// get_custom_logo_image_attributes / get_custom_logo filters.
func GetCustomLogo(c *Context) string {
	if c == nil || !HasCustomLogo(c) {
		htmlOut := ""
		if c != nil && c.Hooks != nil {
			filtered := c.Hooks.ApplyFilters("get_custom_logo", htmlOut, 0)
			if s, ok := filtered.(string); ok {
				return s
			}
		}
		return ""
	}
	id := CustomLogoID(c)
	unlinkHome := CurrentThemeSupports(c, "custom-logo", "unlink-homepage-logo")
	isFront := c.Scope != nil && c.Scope.IsFrontPage
	isPaged := c.Scope != nil && c.Scope.Paged > 1
	isHome := c.Scope != nil && c.Scope.IsHome

	attrs := map[string]string{
		"class":   "custom-logo",
		"loading": "false",
	}
	if unlinkHome && isFront && !isPaged {
		attrs["alt"] = ""
	} else {
		alt := ""
		if c.Meta != nil {
			alt = strings.TrimSpace(fmt.Sprint(c.Meta.GetMetadata(MetaPost, id, "_wp_attachment_image_alt", true)))
			if alt == "<nil>" {
				alt = ""
			}
		}
		if alt == "" {
			attrs["alt"] = optionBlogName(c)
		}
	}
	if c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("get_custom_logo_image_attributes", attrs, id, 0)
		if m, ok := filtered.(map[string]string); ok && m != nil {
			attrs = m
		}
	}

	image := GetAttachmentImage(c, id, "full", false, attrs)
	if strings.TrimSpace(image) == "" {
		return filterCustomLogo(c, "")
	}

	var htmlOut string
	if unlinkHome && isFront && !isPaged {
		htmlOut = `<span class="custom-logo-link">` + image + `</span>`
	} else {
		home := strings.TrimRight(HomeURL(c, "/", ""), "/")
		if home == "" {
			home = "/"
		} else {
			home += "/"
		}
		ariaCurrent := ""
		pageForPosts := int64(0)
		queried := int64(0)
		if c.Options != nil {
			pageForPosts = anyToInt64(c.Options.GetOption("page_for_posts", 0))
		}
		if c.Scope != nil {
			queried = c.Scope.QueriedObjectID
		}
		if !isPaged && (isFront || (isHome && pageForPosts != queried)) {
			ariaCurrent = ` aria-current="page"`
		}
		htmlOut = `<a href="` + html.EscapeString(home) + `" class="custom-logo-link" rel="home"` + ariaCurrent + `>` + image + `</a>`
	}
	return filterCustomLogo(c, htmlOut)
}

func filterCustomLogo(c *Context, htmlOut string) string {
	if c == nil || c.Hooks == nil {
		return htmlOut
	}
	filtered := c.Hooks.ApplyFilters("get_custom_logo", htmlOut, 0)
	if s, ok := filtered.(string); ok {
		return s
	}
	return htmlOut
}

func optionBlogName(c *Context) string {
	if c == nil || c.Options == nil {
		return ""
	}
	v := c.Options.GetOption("blogname", "")
	switch t := v.(type) {
	case string:
		return t
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		if s == "<nil>" {
			return ""
		}
		return s
	}
}

// SetCustomLogoID persists theme_mod custom_logo (test/helper, P3.111).
func SetCustomLogoID(c *Context, id int64) error {
	if c == nil || c.Options == nil {
		return fmt.Errorf("wp: SetCustomLogoID: invalid state")
	}
	mods := loadThemeMods(c)
	if id <= 0 {
		delete(mods, "custom_logo")
	} else {
		mods["custom_logo"] = id
	}
	return saveThemeMods(c, mods)
}

// CustomLogoIDFromOption parses site_logo / custom_logo option values.
func CustomLogoIDFromOption(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return anyToInt64(v)
	}
}
