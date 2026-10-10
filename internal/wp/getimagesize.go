package wp

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// Image type constants matching PHP's IMAGETYPE_* (subset used by wp_getimagesize).
const (
	ImageTypeGIF  = 1
	ImageTypeJPEG = 2
	ImageTypePNG  = 3
	ImageTypeBMP  = 6
	ImageTypeICO  = 17
	ImageTypeWEBP = 18
	ImageTypeAVIF = 19
	ImageTypeHEIC = 20 // PHP uses IMAGETYPE_HEIF (for heic/heif)
)

// ImageSizeInfo is the wp_getimagesize return shape (numeric indices + mime).
type ImageSizeInfo struct {
	Width  int
	Height int
	Type   int
	Attr   string
	Mime   string
}

// GetImagesize returns image dimensions for a file path (wp_getimagesize).
// Returns nil when the file is missing, not an image, or unreadable.
func GetImagesize(filename string) *ImageSizeInfo {
	if filename == "" {
		return nil
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer f.Close()

	cfg, format, err := image.DecodeConfig(f)
	if err == nil && cfg.Width > 0 && cfg.Height > 0 {
		mime, typ := mimeAndTypeFromFormat(format)
		if mime == "" {
			mime = GetImageMime(filename)
			typ = imageTypeFromMime(mime)
		}
		return &ImageSizeInfo{
			Width:  cfg.Width,
			Height: cfg.Height,
			Type:   typ,
			Attr:   fmt.Sprintf(`width="%d" height="%d"`, cfg.Width, cfg.Height),
			Mime:   mime,
		}
	}

	// Fallbacks for formats stdlib DecodeConfig may not handle (WebP/AVIF).
	mime := GetImageMime(filename)
	if mime == "" {
		return nil
	}
	switch mime {
	case "image/webp":
		w, h := webpDimensions(filename)
		if w <= 0 || h <= 0 {
			return nil
		}
		return &ImageSizeInfo{
			Width: w, Height: h, Type: ImageTypeWEBP,
			Attr: fmt.Sprintf(`width="%d" height="%d"`, w, h),
			Mime: "image/webp",
		}
	case "image/avif":
		// AVIF size parsing needs a dedicated parser; without it return nil
		// (matches WP when getimagesize + avif helpers fail).
		return nil
	default:
		return nil
	}
}

func mimeAndTypeFromFormat(format string) (string, int) {
	switch format {
	case "jpeg":
		return "image/jpeg", ImageTypeJPEG
	case "png":
		return "image/png", ImageTypePNG
	case "gif":
		return "image/gif", ImageTypeGIF
	case "bmp":
		return "image/bmp", ImageTypeBMP
	default:
		return "", 0
	}
}

func imageTypeFromMime(mime string) int {
	switch mime {
	case "image/jpeg":
		return ImageTypeJPEG
	case "image/png":
		return ImageTypePNG
	case "image/gif":
		return ImageTypeGIF
	case "image/bmp":
		return ImageTypeBMP
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ImageTypeICO
	case "image/webp":
		return ImageTypeWEBP
	case "image/avif":
		return ImageTypeAVIF
	case "image/heic", "image/heif":
		return ImageTypeHEIC
	default:
		return 0
	}
}

// webpDimensions parses VP8/VP8L/VP8X headers (wp_get_webp_info subset).
func webpDimensions(filename string) (width, height int) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	buf := make([]byte, 40)
	n, err := f.Read(buf)
	if err != nil || n < 30 {
		return 0, 0
	}
	buf = buf[:n]
	if n < 16 || string(buf[0:4]) != "RIFF" || string(buf[8:12]) != "WEBP" {
		return 0, 0
	}
	kind := string(buf[12:16])
	switch kind {
	case "VP8 ":
		if n < 30 {
			return 0, 0
		}
		width = int(buf[26]) | int(buf[27])<<8
		height = int(buf[28]) | int(buf[29])<<8
		width &= 0x3FFF
		height &= 0x3FFF
	case "VP8L":
		if n < 25 {
			return 0, 0
		}
		b0, b1, b2, b3 := buf[21], buf[22], buf[23], buf[24]
		width = int(b0) | int(b1&0x3F)<<8
		height = int((b1&0xC0)>>6) | int(b2)<<2 | int(b3&0x03)<<10
		width++
		height++
	case "VP8X":
		if n < 30 {
			return 0, 0
		}
		width = int(buf[24]) | int(buf[25])<<8 | int(buf[26])<<16
		height = int(buf[27]) | int(buf[28])<<8 | int(buf[29])<<16
		width++
		height++
	}
	return width, height
}
