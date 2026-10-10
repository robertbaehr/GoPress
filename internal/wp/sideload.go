// Media sideload helpers (P8.27) — wp_handle_sideload / media_handle_sideload subset.
// Remote download_url is intentionally not implemented (no egress).
package wp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SideloadFile mirrors the PHP $file_array passed to wp_handle_sideload.
type SideloadFile struct {
	Name    string // original filename
	TmpName string // absolute path to a local temporary file
}

// SideloadResult is the success shape of wp_handle_sideload (file/url/type).
type SideloadResult struct {
	File string // absolute filesystem path
	URL  string
	Type string // mime type
}

// HandleSideload moves a local temp file into uploads (wp_handle_sideload subset).
// Does not create an attachment row — use MediaHandleSideload for that.
func HandleSideload(c *Context, file SideloadFile) (*SideloadResult, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: HandleSideload: nil context")
	}
	name := sanitizeUploadName(filepath.Base(strings.ReplaceAll(file.Name, "\\", "/")))
	if name == "" || name == "." || name == ".." {
		return nil, fmt.Errorf("upload_error: empty filename")
	}
	tmp := strings.TrimSpace(file.TmpName)
	if tmp == "" {
		return nil, fmt.Errorf("upload_error: empty tmp_name")
	}
	st, err := os.Stat(tmp)
	if err != nil || st.IsDir() {
		return nil, fmt.Errorf("upload_error: tmp file missing")
	}
	preCheck := CheckFiletype(c, name, nil)
	if preCheck.Ext == "" || preCheck.Type == "" {
		return nil, fmt.Errorf("upload_error: sorry, this file type is not permitted for security reasons")
	}

	uploads := GetUploadDir(c)
	if uploads.Basedir == "" {
		return nil, fmt.Errorf("upload_error: empty upload dir")
	}
	dir := filepath.Join(uploads.Basedir, filepath.FromSlash(UploadFixedSubdir))
	if !MkdirP(dir) {
		return nil, fmt.Errorf("upload_error: mkdir %s", dir)
	}
	destName := UniqueFilename(dir, name)
	abs := filepath.Join(dir, destName)

	// Prefer rename; fall back to copy+remove (cross-device).
	if err := os.Rename(tmp, abs); err != nil {
		if copyErr := copyFile(tmp, abs); copyErr != nil {
			return nil, fmt.Errorf("upload_error: %v", copyErr)
		}
		_ = os.Remove(tmp)
	}

	checked := CheckFiletypeAndExt(c, abs, destName, nil)
	if checked.Ext == "" || checked.Type == "" {
		_ = os.Remove(abs)
		return nil, fmt.Errorf("upload_error: sorry, this file type is not permitted for security reasons")
	}
	if checked.ProperFilename != "" {
		proper := sanitizeUploadName(filepath.Base(checked.ProperFilename))
		if proper != "" && proper != destName {
			newAbs := filepath.Join(dir, proper)
			if err := os.Rename(abs, newAbs); err == nil {
				abs = newAbs
				destName = proper
			}
		}
	}
	rel := UploadFixedSubdir + "/" + destName
	url := strings.TrimRight(uploads.Baseurl, "/") + "/" + rel
	return &SideloadResult{File: abs, URL: url, Type: checked.Type}, nil
}

// MediaHandleSideload creates an attachment from a sideloaded temp file (media_handle_sideload).
// desc, when non-empty, becomes the attachment title.
func MediaHandleSideload(c *Context, file SideloadFile, postID int64, desc string) (*UploadedFile, error) {
	moved, err := HandleSideload(c, file)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSuffix(filepath.Base(moved.File), filepath.Ext(moved.File))
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")
	content := ""
	alt := ""
	if strings.HasPrefix(strings.ToLower(moved.Type), "image/") {
		if meta, ok := ReadImageMetadata(c, moved.File); ok && meta != nil {
			if t := strings.TrimSpace(fmt.Sprint(meta["title"])); t != "" && t != "<nil>" && !isNumericSanitizeTitle(t) {
				title = t
			}
			if cap := strings.TrimSpace(fmt.Sprint(meta["caption"])); cap != "" && cap != "<nil>" {
				content = cap
			}
			if a := strings.TrimSpace(fmt.Sprint(meta["alt"])); a != "" && a != "<nil>" {
				alt = a
			}
		}
	}
	if d := strings.TrimSpace(desc); d != "" {
		title = d
	}

	author := int64(0)
	if c.CurrentUser != nil {
		author = c.CurrentUser.ID
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	id, err := insertAttachmentRowParent(c, author, now, title, SanitizeTitle(title), content, moved.Type, moved.URL, postID)
	if err != nil {
		_ = os.Remove(moved.File)
		return nil, err
	}
	rel := UploadFixedSubdir + "/" + filepath.Base(moved.File)
	if c.Meta != nil {
		_, _, _ = c.Meta.AddMetadata(MetaPost, id, "_wp_attached_file", rel, true)
		if alt != "" {
			_, _, _ = c.Meta.AddMetadata(MetaPost, id, "_wp_attachment_image_alt", alt, true)
		}
		if strings.HasPrefix(strings.ToLower(moved.Type), "image/") {
			meta := GenerateAttachmentMetadata(c, id, moved.File)
			if meta != nil {
				meta["file"] = rel
				_ = StoreAttachmentMetadata(c, id, meta)
			}
		}
	}
	return &UploadedFile{
		ID:       id,
		RelPath:  rel,
		AbsPath:  moved.File,
		URL:      moved.URL,
		MimeType: moved.Type,
		Title:    title,
	}, nil
}

// MediaSideloadImage is media_sideload_image without remote download (P8.27).
// Only local filesystem paths are accepted; http(s) URLs return an error.
// returnType is "html" (default), "src", or "id".
func MediaSideloadImage(c *Context, file string, postID int64, desc, returnType string) (any, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return nil, fmt.Errorf("image_sideload_failed: invalid image URL")
	}
	lower := strings.ToLower(file)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "//") {
		return nil, fmt.Errorf("image_sideload_failed: remote download is not available")
	}
	// Strip optional file:// prefix.
	path := strings.TrimPrefix(file, "file://")
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return nil, fmt.Errorf("image_sideload_failed: invalid image URL")
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	allowed := map[string]bool{"jpg": true, "jpeg": true, "jpe": true, "png": true, "gif": true, "webp": true}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("image_sideload_extensions", []string{"jpg", "jpeg", "jpe", "png", "gif", "webp"}); v != nil {
			if list, ok := v.([]string); ok {
				allowed = map[string]bool{}
				for _, e := range list {
					allowed[strings.ToLower(strings.TrimSpace(e))] = true
				}
			}
		}
	}
	if !allowed[ext] {
		return nil, fmt.Errorf("image_sideload_failed: Invalid image URL.")
	}
	// Copy to a temp file so HandleSideload can consume/move it.
	tmp, err := copyToTemp(path, filepath.Base(path))
	if err != nil {
		return nil, fmt.Errorf("image_sideload_failed: %v", err)
	}
	up, err := MediaHandleSideload(c, SideloadFile{Name: filepath.Base(path), TmpName: tmp}, postID, desc)
	if err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(returnType)) {
	case "src":
		return up.URL, nil
	case "id":
		return up.ID, nil
	default:
		alt := htmlEscapeAttr(desc)
		if alt == "" {
			alt = htmlEscapeAttr(up.Title)
		}
		return fmt.Sprintf(`<img src="%s" alt="%s" />`, htmlEscapeAttr(up.URL), alt), nil
	}
}

func insertAttachmentRowParent(c *Context, author int64, now, title, slug, content, mimeType, guid string, parent int64) (int64, error) {
	if slug == "" {
		slug = "attachment"
	}
	if parent < 0 {
		parent = 0
	}
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		author, now, now, content, title, "", "inherit",
		"open", "closed", "", slug, "", "", now, now,
		"", parent, guid, 0, "attachment", mimeType, 0,
	)
	if err != nil {
		return 0, err
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dst)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dst)
		return closeErr
	}
	return nil
}

func copyToTemp(src, name string) (string, error) {
	tmpDir := os.TempDir()
	dest := filepath.Join(tmpDir, "gopress-sideload-"+SanitizeFileName(name))
	// Ensure uniqueness.
	dest = UniqueFilename(tmpDir, "gopress-sideload-"+SanitizeFileName(name))
	dest = filepath.Join(tmpDir, dest)
	if err := copyFile(src, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func isNumericSanitizeTitle(s string) bool {
	s = strings.TrimSpace(SanitizeTitle(s))
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
