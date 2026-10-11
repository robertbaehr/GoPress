package wp

import (
	"regexp"
	"strconv"
	"strings"
)

var reHTMLClassStripPct = regexp.MustCompile(`%[a-fA-F0-9]{2}`)
var reHTMLClassKeep = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// SanitizeHTMLClass mirrors sanitize_html_class (P3.116).
func SanitizeHTMLClass(classname, fallback string) string {
	sanitized := reHTMLClassStripPct.ReplaceAllString(classname, "")
	sanitized = reHTMLClassKeep.ReplaceAllString(sanitized, "")
	if sanitized == "" && fallback != "" {
		return SanitizeHTMLClass(fallback, "")
	}
	return sanitized
}

// CommentClassState tracks WP globals $comment_alt / $comment_depth / $comment_thread_alt.
type CommentClassState struct {
	Alt       int
	Depth     int
	ThreadAlt int
}

// GetCommentClass mirrors get_comment_class() (P3.116).
// state may be nil (depth defaults to 1; alt/threadAlt start at 0 and advance).
func GetCommentClass(c *Context, commentID int64, postID int64, state *CommentClassState, cssClass ...string) []string {
	extra := splitPostClasses(cssClass...)
	if c == nil || c.DB == nil || commentID <= 0 {
		return extra
	}
	cm, err := GetCommentByID(c, commentID)
	if err != nil || cm == nil {
		return extra
	}
	if state == nil {
		state = &CommentClassState{Depth: 1}
	}
	if state.Depth <= 0 {
		state.Depth = 1
	}

	ctype := strings.TrimSpace(cm.Type)
	if ctype == "" {
		ctype = "comment"
	}
	classes := []string{ctype}

	if cm.UserID > 0 {
		classes = append(classes, "byuser")
		nicename := ""
		if u, err := c.GetUserBy("id", strconv.FormatInt(cm.UserID, 10)); err == nil && u != nil {
			nicename = strings.TrimSpace(u.Nicename)
		}
		if nicename == "" {
			nicename = strconv.FormatInt(cm.UserID, 10)
		}
		classes = append(classes, "comment-author-"+SanitizeHTMLClass(nicename, strconv.FormatInt(cm.UserID, 10)))
		pid := postID
		if pid <= 0 {
			pid = cm.PostID
		}
		if p, err := GetPostByID(c, pid); err == nil && p != nil && p.Author == cm.UserID {
			classes = append(classes, "bypostauthor")
		}
	}

	if state.Alt%2 != 0 {
		classes = append(classes, "odd", "alt")
	} else {
		classes = append(classes, "even")
	}
	state.Alt++

	if state.Depth == 1 {
		if state.ThreadAlt%2 != 0 {
			classes = append(classes, "thread-odd", "thread-alt")
		} else {
			classes = append(classes, "thread-even")
		}
		state.ThreadAlt++
	}

	classes = append(classes, "depth-"+strconv.Itoa(state.Depth))
	classes = append(classes, extra...)
	classes = dedupePostClasses(classes)

	if c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("comment_class", classes, extra, strconv.FormatInt(cm.ID, 10), cm, postID)
		switch t := filtered.(type) {
		case []string:
			classes = t
		case []any:
			classes = make([]string, 0, len(t))
			for _, v := range t {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					classes = append(classes, strings.TrimSpace(s))
				}
			}
		}
		classes = dedupePostClasses(classes)
	}
	return classes
}

// BuildCommentClass builds get_comment_class output from already-loaded fields (P3.116).
// Used by core/comment-template without a live Context.
func BuildCommentClass(commentType string, userID int64, userNicename string, postAuthor int64, depth, alt, threadAlt int, cssClass ...string) (classes []string, nextAlt, nextThreadAlt int) {
	if strings.TrimSpace(commentType) == "" {
		commentType = "comment"
	}
	if depth <= 0 {
		depth = 1
	}
	classes = []string{commentType}
	if userID > 0 {
		classes = append(classes, "byuser")
		nicename := strings.TrimSpace(userNicename)
		if nicename == "" {
			nicename = strconv.FormatInt(userID, 10)
		}
		classes = append(classes, "comment-author-"+SanitizeHTMLClass(nicename, strconv.FormatInt(userID, 10)))
		if postAuthor > 0 && userID == postAuthor {
			classes = append(classes, "bypostauthor")
		}
	}
	if alt%2 != 0 {
		classes = append(classes, "odd", "alt")
	} else {
		classes = append(classes, "even")
	}
	nextAlt = alt + 1
	nextThreadAlt = threadAlt
	if depth == 1 {
		if threadAlt%2 != 0 {
			classes = append(classes, "thread-odd", "thread-alt")
		} else {
			classes = append(classes, "thread-even")
		}
		nextThreadAlt = threadAlt + 1
	}
	classes = append(classes, "depth-"+strconv.Itoa(depth))
	classes = append(classes, splitPostClasses(cssClass...)...)
	return dedupePostClasses(classes), nextAlt, nextThreadAlt
}
