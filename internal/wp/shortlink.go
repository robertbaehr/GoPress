package wp

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// GetShortlink returns a ?p= shortlink for a post (wp_get_shortlink subset, P3.40 / P3.106).
// context "post" uses id; context "query" uses queriedID when id is 0.
// Static front pages return the home URL.
// P3.106: pre_get_shortlink / get_shortlink filters; allow_slugs arg (unused by default).
func GetShortlink(c *Context, id int64, context string, queriedID int64) string {
	return GetShortlinkAllowSlugs(c, id, context, queriedID, true)
}

// GetShortlinkAllowSlugs is wp_get_shortlink with $allow_slugs (P3.106).
// Core shortlinks ignore allowSlugs; filters receive it for plugin parity.
func GetShortlinkAllowSlugs(c *Context, id int64, context string, queriedID int64, allowSlugs bool) string {
	ctx := strings.ToLower(strings.TrimSpace(context))
	if ctx == "" {
		ctx = "post"
	}
	// pre_get_shortlink: non-false short-circuits (WP returns early when !== false).
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("pre_get_shortlink", false, id, ctx, allowSlugs)
		switch t := filtered.(type) {
		case bool:
			if t {
				// true is non-false in PHP loose sense but WP checks !== false;
				// treat bare true as empty shortlink short-circuit.
				return getShortlinkFilter(c, "", id, ctx, allowSlugs)
			}
			// false → continue generation
		case string:
			return getShortlinkFilter(c, t, id, ctx, allowSlugs)
		default:
			if filtered != nil && filtered != false {
				s := strings.TrimSpace(fmt.Sprint(filtered))
				if s != "" && s != "<nil>" && s != "false" {
					return getShortlinkFilter(c, s, id, ctx, allowSlugs)
				}
			}
		}
	}

	postID := id
	if ctx == "query" && postID <= 0 {
		postID = queriedID
	}
	shortlink := ""
	if postID > 0 {
		post, err := GetPostByID(c, postID)
		if err == nil && post != nil {
			switch post.Type {
			case "post", "page", "attachment":
				// public shortlink types (WP: post_type->public)
			default:
				post = nil
			}
			if post != nil {
				if post.Type == "page" && c != nil && c.Options != nil {
					show := optionValueString(c.Options.GetOption("show_on_front", "posts"))
					front := optionValueString(c.Options.GetOption("page_on_front", "0"))
					frontID, _ := strconv.ParseInt(strings.TrimSpace(front), 10, 64)
					if show == "page" && frontID == post.ID {
						home := strings.TrimRight(HomeURL(c, "/", ""), "/")
						shortlink = home + "/"
					}
				}
				if shortlink == "" {
					// Mirror home_url( '?p=' . $post_id ) — slash before ?.
					home := strings.TrimRight(HomeURL(c, "", ""), "/")
					if home != "" {
						shortlink = fmt.Sprintf("%s/?p=%d", home, post.ID)
					}
				}
			}
		}
	}
	_ = allowSlugs // core default ignores slugs; exposed for filters.
	return getShortlinkFilter(c, shortlink, id, ctx, allowSlugs)
}

func getShortlinkFilter(c *Context, shortlink string, id int64, context string, allowSlugs bool) string {
	if c == nil || c.Hooks == nil {
		return shortlink
	}
	filtered := c.Hooks.ApplyFilters("get_shortlink", shortlink, id, context, allowSlugs)
	if s, ok := filtered.(string); ok {
		return s
	}
	return shortlink
}

// ShortlinkHeader emits Link: <url>; rel=shortlink (wp_shortlink_header, P3.130).
// Uses wp_get_shortlink(0, 'query') with the queried object ID.
func ShortlinkHeader(w http.ResponseWriter, c *Context, queriedID int64) {
	if w == nil {
		return
	}
	short := GetShortlink(c, 0, "query", queriedID)
	if strings.TrimSpace(short) == "" {
		return
	}
	w.Header().Add("Link", fmt.Sprintf(`<%s>; rel=shortlink`, short))
}
