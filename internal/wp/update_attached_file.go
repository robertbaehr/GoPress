package wp

import (
	"strings"
)

// UpdateAttachedFile updates _wp_attached_file for an attachment
// (update_attached_file, P8.55). Applies update_attached_file filter, then
// stores a path relative to uploads basedir when possible. Empty relative
// path deletes the meta. Returns false when the attachment is missing.
func UpdateAttachedFile(c *Context, attachmentID int64, file string) bool {
	if c == nil || c.Meta == nil || attachmentID <= 0 {
		return false
	}
	post, err := GetPost(c, QueryArgs{P: attachmentID, PostType: "attachment", PostStatus: "inherit"})
	if err != nil || post == nil {
		return false
	}
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("update_attached_file", file, attachmentID); v != nil {
			if s, ok := v.(string); ok {
				file = s
			}
		}
	}
	rel := RelativeUploadPath(c, file)
	if strings.TrimSpace(rel) == "" {
		_, _ = c.Meta.DeleteMetadata(MetaPost, attachmentID, "_wp_attached_file", "")
		return true
	}
	_, _, _, err = c.Meta.UpdateMetadata(MetaPost, attachmentID, "_wp_attached_file", rel, "")
	return err == nil
}

// RelativeUploadPath returns path relative to uploads basedir
// (_wp_relative_upload_path, P8.55). Unchanged when outside basedir.
func RelativeUploadPath(c *Context, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	uploads := GetUploadDir(c)
	base := strings.TrimRight(strings.ReplaceAll(uploads.Basedir, "\\", "/"), "/")
	norm := strings.ReplaceAll(path, "\\", "/")
	if base != "" && strings.HasPrefix(norm, base+"/") {
		return strings.TrimPrefix(norm, base+"/")
	}
	if base != "" && norm == base {
		return ""
	}
	return strings.TrimPrefix(norm, "/")
}
