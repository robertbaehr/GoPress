package wp

import (
	"bytes"
	"os"
)

// GetImageMime returns the MIME type of an image file by inspecting contents
// (wp_get_image_mime). Returns "" when the file is missing or not a recognized image.
func GetImageMime(file string) string {
	if file == "" {
		return ""
	}
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()

	hdr := make([]byte, 16)
	n, err := f.Read(hdr)
	if err != nil || n < 3 {
		return ""
	}
	hdr = hdr[:n]

	if mime := imageMimeFromMagic(hdr); mime != "" {
		return mime
	}
	return ""
}

func imageMimeFromMagic(b []byte) string {
	if len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff {
		return "image/jpeg"
	}
	if len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return "image/png"
	}
	if len(b) >= 6 && (bytes.Equal(b[:6], []byte("GIF87a")) || bytes.Equal(b[:6], []byte("GIF89a"))) {
		return "image/gif"
	}
	// BMP
	if len(b) >= 2 && b[0] == 'B' && b[1] == 'M' {
		return "image/bmp"
	}
	// ICO
	if len(b) >= 4 && b[0] == 0 && b[1] == 0 && b[2] == 1 && b[3] == 0 {
		return "image/x-icon"
	}
	// WebP: RIFF....WEBP
	if len(b) >= 12 && bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")) {
		return "image/webp"
	}
	// ISO BMFF (AVIF/HEIC/HEIF): ....ftypXXXX
	if len(b) >= 12 && bytes.Equal(b[4:8], []byte("ftyp")) {
		brand := string(b[8:12])
		switch brand {
		case "avif", "avis":
			return "image/avif"
		case "heic":
			return "image/heic"
		case "heif":
			return "image/heif"
		}
	}
	return ""
}
