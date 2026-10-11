package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// wantsRESTEmbed reports whether the request asked for linked embeds (P4.146).
// Mirrors WP REST Server::_embed: presence of _embed (any value) enables embeds;
// comma/space-separated values restrict which relations are included.
func wantsRESTEmbed(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	_, ok := r.URL.Query()["_embed"]
	return ok
}

// parseEmbedRelations returns the set of embed relations to include.
// Empty set means "all embeddable relations" (P4.146).
func parseEmbedRelations(r *http.Request) map[string]bool {
	if r == nil || r.URL == nil {
		return nil
	}
	vals := r.URL.Query()["_embed"]
	if len(vals) == 0 {
		return nil
	}
	out := map[string]bool{}
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" || v == "1" || strings.EqualFold(v, "true") {
			continue
		}
		for _, part := range strings.FieldsFunc(v, func(r rune) bool {
			return r == ',' || r == ' '
		}) {
			part = strings.TrimSpace(part)
			if part != "" {
				out[part] = true
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func embedRelationWanted(wanted map[string]bool, rel string) bool {
	if wanted == nil {
		return true
	}
	return wanted[rel]
}

// applyPostsRESTEmbed attaches _embedded for ?_embed (P4.146).
func applyPostsRESTEmbed(c *wp.Context, posts []Post, r *http.Request) {
	if !wantsRESTEmbed(r) {
		return
	}
	wanted := parseEmbedRelations(r)
	for i := range posts {
		posts[i].Embedded = buildPostEmbedded(c, &posts[i], wanted)
	}
}

func applyPostRESTEmbed(c *wp.Context, post *Post, r *http.Request) {
	if post == nil || !wantsRESTEmbed(r) {
		return
	}
	post.Embedded = buildPostEmbedded(c, post, parseEmbedRelations(r))
}

// buildPostEmbedded resolves embeddable _links into _embedded (P4.146 / P4.148).
// Supports author, wp:featuredmedia, wp:term, replies, and up (page parent).
func buildPostEmbedded(c *wp.Context, post *Post, wanted map[string]bool) map[string]any {
	if post == nil {
		return nil
	}
	out := map[string]any{}
	if embedRelationWanted(wanted, "author") {
		if author := embedAuthor(c, post.Author); author != nil {
			out["author"] = []any{author}
		}
	}
	if embedRelationWanted(wanted, "wp:featuredmedia") && post.FeaturedMedia > 0 {
		if m, err := getMediaByID(c, post.FeaturedMedia); err == nil && m != nil {
			out["wp:featuredmedia"] = []any{*m}
		}
	}
	if embedRelationWanted(wanted, "wp:term") && post.Type == "post" {
		terms := embedPostTerms(c, post)
		if len(terms) > 0 {
			out["wp:term"] = terms
		}
	}
	// P4.148: replies → approved comments in embed context.
	if embedRelationWanted(wanted, "replies") {
		if replies := embedPostReplies(c, post.ID); replies != nil {
			out["replies"] = replies
		}
	}
	// P4.148: up → parent page (pages only; embeddable when Parent > 0).
	if embedRelationWanted(wanted, "up") && post.Type == "page" && post.Parent > 0 {
		if parent := embedParentPage(c, post.Parent); parent != nil {
			out["up"] = []any{parent}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// embedPostReplies returns [[comments…]] matching WP replies embed shape (P4.148).
func embedPostReplies(c *wp.Context, postID int64) []any {
	if c == nil || postID <= 0 {
		return nil
	}
	comments, err := listCommentsPaged(c, commentListQuery{
		PostID: postID,
		Status: commentStatusFilter{Approved: []string{"1"}},
	}, 100, 1)
	if err != nil || len(comments) == 0 {
		return nil
	}
	items := make([]any, 0, len(comments))
	for i := range comments {
		cm := comments[i]
		cm.RESTContext = "embed"
		items = append(items, cm)
	}
	return []any{items}
}

// embedParentPage returns the embed-context parent page (P4.148).
func embedParentPage(c *wp.Context, parentID int64) *Post {
	if c == nil || parentID <= 0 {
		return nil
	}
	p, err := getPostByID(c, "page", parentID, false)
	if err != nil || p == nil {
		return nil
	}
	p.RESTContext = "embed"
	return p
}

// embedAuthor returns the public/embed user subset for a post author (P4.146).
func embedAuthor(c *wp.Context, authorID int64) *User {
	if c == nil || authorID <= 0 {
		return nil
	}
	u, err := c.GetUserBy("id", authorID)
	if err != nil || u == nil {
		return nil
	}
	listed := buildListedUser(c, u, false, "embed")
	listed.AvatarURLs = gravatarAvatarURLs(u.Email)
	return &listed
}

// embedPostTerms returns [categories[], tags[]] matching WP wp:term embed order.
func embedPostTerms(c *wp.Context, post *Post) []any {
	if c == nil || post == nil {
		return nil
	}
	var cats, tags []any
	for _, id := range post.Categories {
		if t, err := getTermByID(c, "category", id); err == nil && t != nil {
			cats = append(cats, *t)
		}
	}
	for _, id := range post.Tags {
		if t, err := getTermByID(c, "post_tag", id); err == nil && t != nil {
			tags = append(tags, *t)
		}
	}
	if len(cats) == 0 && len(tags) == 0 {
		return nil
	}
	// WP emits one array per wp:term link entry (category then post_tag).
	return []any{cats, tags}
}
