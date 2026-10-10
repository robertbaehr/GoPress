// Media upload helpers (P7.7 / P8.31) — store multipart files under uploads/2024/06.
// Ports wp_handle_upload / media_handle_upload subsets used by async-upload and media-new.
package wp

import (
	"fmt"
	"io"
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

// UploadResult is the success shape of wp_handle_upload (file/url/type) (P8.31).
type UploadResult struct {
	File string // absolute filesystem path
	URL  string
	Type string // mime type
}

// HandleUpload writes a multipart upload into uploads/ (wp_handle_upload subset, P8.31).
// Does not create an attachment row — use MediaHandleUpload for that.
func HandleUpload(c *Context, filename string, r io.Reader, size int64) (*UploadResult, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("upload_error: nil context")
	}
	filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	filename = sanitizeUploadName(filename)
	if filename == "" || filename == "." || filename == ".." {
		return nil, fmt.Errorf("upload_error: empty filename")
	}
	// P8.24: reject filenames outside upload_mimes / get_allowed_mime_types.
	preCheck := CheckFiletype(c, filename, nil)
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
		return nil, fmt.Errorf("upload_error: empty upload")
	}

	// P8.24: resolve mime via CheckFiletypeAndExt (upload_mimes + real image mime).
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
	return &UploadResult{File: abs, URL: url, Type: checked.Type}, nil
}

// MediaHandleUpload creates an attachment from an upload stream (media_handle_upload, P8.31).
// postID becomes post_parent when > 0.
func MediaHandleUpload(c *Context, filename string, r io.Reader, size int64, postID int64) (*UploadedFile, error) {
	moved, err := HandleUpload(c, filename, r, size)
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
		// P8.25: generate attachment metadata (width/height/image_meta) for images.
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

// SaveUploadedMedia writes file content under uploads/2024/06 and creates an attachment post.
// Equivalent to MediaHandleUpload(..., postID=0) (P7.7).
func SaveUploadedMedia(c *Context, filename string, r io.Reader, size int64) (*UploadedFile, error) {
	up, err := MediaHandleUpload(c, filename, r, size, 0)
	if err != nil {
		// Preserve historical error prefix used by callers/tests.
		msg := err.Error()
		if strings.HasPrefix(msg, "upload_error: ") {
			return nil, fmt.Errorf("wp: SaveUploadedMedia: %s", strings.TrimPrefix(msg, "upload_error: "))
		}
		return nil, fmt.Errorf("wp: SaveUploadedMedia: %w", err)
	}
	return up, nil
}

func sanitizeUploadName(name string) string {
	return SanitizeFileName(strings.TrimSpace(name))
}
