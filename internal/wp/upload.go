// Media upload helpers (P7.7) — store multipart files under uploads/2024/06.
package wp

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UploadFixedSubdir is the fixture-aligned uploads subdirectory for new files.
const UploadFixedSubdir = "2024/06"

// UploadedFile is the result of saving an uploaded media file.
type UploadedFile struct {
	ID       int64
	RelPath  string // e.g. 2024/06/photo.png
	AbsPath  string
	URL      string
	MimeType string
	Title    string
}

// SaveUploadedMedia writes file content under uploads/2024/06 and creates an attachment post.
func SaveUploadedMedia(c *Context, filename string, r io.Reader, size int64) (*UploadedFile, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: SaveUploadedMedia: nil context")
	}
	filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	filename = sanitizeUploadName(filename)
	if filename == "" || filename == "." || filename == ".." {
		return nil, fmt.Errorf("wp: SaveUploadedMedia: empty filename")
	}

	uploads := GetUploadDir(c)
	if uploads.Basedir == "" {
		return nil, fmt.Errorf("wp: SaveUploadedMedia: empty upload dir")
	}
	dir := filepath.Join(uploads.Basedir, filepath.FromSlash(UploadFixedSubdir))
	if !MkdirP(dir) {
		return nil, fmt.Errorf("wp: SaveUploadedMedia: mkdir %s", dir)
	}

	destName := UniqueFilename(dir, filename)
	abs := filepath.Join(dir, destName)
	f, err := os.OpenFile(abs, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	n, copyErr := io.Copy(f, r)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(abs)
		return nil, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(abs)
		return nil, closeErr
	}
	if size > 0 && n == 0 {
		_ = os.Remove(abs)
		return nil, fmt.Errorf("wp: SaveUploadedMedia: empty upload")
	}

	mimeType := mimeFromFilename(destName)
	rel := UploadFixedSubdir + "/" + destName
	url := strings.TrimRight(uploads.Baseurl, "/") + "/" + rel
	title := strings.TrimSuffix(destName, filepath.Ext(destName))
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	author := int64(0)
	if c.CurrentUser != nil {
		author = c.CurrentUser.ID
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	id, err := insertAttachmentRow(c, author, now, title, SanitizeTitle(title), mimeType, url)
	if err != nil {
		_ = os.Remove(abs)
		return nil, err
	}
	if c.Meta != nil {
		_, _, _ = c.Meta.AddMetadata(MetaPost, id, "_wp_attached_file", rel, true)
	}
	return &UploadedFile{
		ID:       id,
		RelPath:  rel,
		AbsPath:  abs,
		URL:      url,
		MimeType: mimeType,
		Title:    title,
	}, nil
}

func insertAttachmentRow(c *Context, author int64, now, title, slug, mimeType, guid string) (int64, error) {
	if slug == "" {
		slug = "attachment"
	}
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		author, now, now, "", title, "", "inherit",
		"open", "closed", "", slug, "", "", now, now,
		"", 0, guid, 0, "attachment", mimeType, 0,
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

func sanitizeUploadName(name string) string {
	return SanitizeFileName(strings.TrimSpace(name))
}

func mimeFromFilename(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return "application/octet-stream"
	}
	if mt := mime.TypeByExtension(ext); mt != "" {
		return mt
	}
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".mp3":
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}
