package wp

import (
	"fmt"
	"html"
	"strings"
)

// GetPreviousImageLink returns the previous sibling image link
// (get_previous_image_link, P8.59).
func GetPreviousImageLink(c *Context, attachmentID int64, size, text string) string {
	return GetAdjacentImageLink(c, attachmentID, true, size, text)
}

// GetNextImageLink returns the next sibling image link
// (get_next_image_link, P8.59).
func GetNextImageLink(c *Context, attachmentID int64, size, text string) string {
	return GetAdjacentImageLink(c, attachmentID, false, size, text)
}

// GetAdjacentImageLink returns the previous or next image attachment link
// among siblings that share the same post_parent (get_adjacent_image_link, P8.59).
// Applies previous_image_link / next_image_link filters.
func GetAdjacentImageLink(c *Context, attachmentID int64, prev bool, size, text string) string {
	if c == nil || attachmentID <= 0 {
		return ""
	}
	post, err := GetPost(c, QueryArgs{P: attachmentID, PostType: "attachment", PostStatus: "inherit"})
	if err != nil || post == nil {
		return ""
	}
	if size == "" {
		size = "thumbnail"
	}
	siblings, err := GetPosts(c, QueryArgs{
		PostType:     "attachment",
		PostStatus:   "inherit",
		ParentIn:     []int64{post.Parent},
		OrderBy:      "menu_order",
		Order:        "ASC",
		PostsPerPage: -1,
	})
	if err != nil {
		return ""
	}
	var images []Post
	for _, s := range siblings {
		mime := GetPostMimeTypeString(c, s.ID)
		if mimeMatches(mime, "image") {
			images = append(images, s)
		}
	}
	idx := -1
	for i, img := range images {
		if img.ID == attachmentID {
			idx = i
			break
		}
	}
	output := ""
	var adjID int64
	if idx >= 0 {
		next := idx + 1
		if prev {
			next = idx - 1
		}
		if next >= 0 && next < len(images) {
			adjID = images[next].ID
			if strings.TrimSpace(text) != "" {
				url := ""
				if home := strings.TrimRight(HomeURL(c, "/", ""), "/"); home != "" {
					url = fmt.Sprintf("%s/?attachment_id=%d", home, adjID)
				}
				if url == "" {
					url = GetAttachmentURL(c, adjID)
				}
				if url != "" {
					output = `<a href="` + html.EscapeString(url) + `">` + html.EscapeString(text) + `</a>`
				}
			} else {
				output = GetAttachmentLink(c, adjID, size, true)
			}
		}
	}
	hook := "next_image_link"
	if prev {
		hook = "previous_image_link"
	}
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters(hook, output, adjID, size, text); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return output
}
