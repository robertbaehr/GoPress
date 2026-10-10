package wp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GetCanonicalURL returns the canonical URL for a published post
// (wp_get_canonical_url subset, P3.42). page/cpage mirror query vars for
// multipage content and comment pagination when > 1 / > 0 respectively.
func GetCanonicalURL(c *Context, postID int64, page, cpage int) string {
	if postID <= 0 {
		return ""
	}
	post, err := GetPostByID(c, postID)
	if err != nil || post == nil {
		return ""
	}
	if !strings.EqualFold(strings.TrimSpace(post.Status), "publish") {
		return ""
	}
	url := postCanonicalPermalink(c, post)
	if url == "" {
		return ""
	}
	if page >= 2 {
		structure := ""
		if c != nil && c.Options != nil {
			structure = strings.TrimSpace(fmt.Sprint(c.Options.GetOption("permalink_structure", "")))
		}
		if structure == "" {
			url = addQueryArg(url, "page", strconv.Itoa(page))
		} else {
			url = strings.TrimRight(url, "/") + "/" + strconv.Itoa(page) + "/"
		}
	}
	if cpage > 0 {
		url = addQueryArg(url, "cpage", strconv.Itoa(cpage))
	}
	return url
}

// RelCanonicalLink returns <link rel="canonical" …> HTML for a post, or "".
func RelCanonicalLink(c *Context, postID int64) string {
	url := GetCanonicalURL(c, postID, 0, 0)
	if url == "" {
		return ""
	}
	return `<link rel="canonical" href="` + htmlEscapeAttr(url) + `" />` + "\n"
}

func postCanonicalPermalink(c *Context, post *Post) string {
	home := strings.TrimRight(HomeURL(c, "/", ""), "/")
	if home == "" {
		return ""
	}
	if post.Type == "page" && post.Name != "" {
		show := ""
		front := ""
		if c != nil && c.Options != nil {
			show = optionValueString(c.Options.GetOption("show_on_front", "posts"))
			front = optionValueString(c.Options.GetOption("page_on_front", "0"))
		}
		frontID, _ := strconv.ParseInt(strings.TrimSpace(front), 10, 64)
		if show == "page" && frontID == post.ID {
			return home + "/"
		}
		return home + "/" + post.Name + "/"
	}
	if post.Type == "post" && post.Name != "" && post.Date != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", post.Date); err == nil {
			return fmt.Sprintf("%s/%04d/%02d/%02d/%s/", home, t.Year(), int(t.Month()), t.Day(), post.Name)
		}
	}
	return fmt.Sprintf("%s/?p=%d", home, post.ID)
}

func addQueryArg(rawURL, key, value string) string {
	if strings.Contains(rawURL, "?") {
		return rawURL + "&" + key + "=" + value
	}
	return rawURL + "?" + key + "=" + value
}

func htmlEscapeAttr(s string) string {
	replacer := strings.NewReplacer(
		`&`, `&amp;`,
		`"`, `&quot;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
	)
	return replacer.Replace(s)
}
