package wp

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var reImageEditHash = regexp.MustCompile(`-e[0-9]{13}`)

// GetAttachmentImageSrcset is wp_get_attachment_image_srcset (P8.45).
// Returns "" when unavailable (PHP false).
func GetAttachmentImageSrcset(c *Context, attachmentID int64, size string, imageMeta map[string]any) string {
	src, ok := GetAttachmentImageSrc(c, attachmentID, size, false)
	if !ok {
		return ""
	}
	if imageMeta == nil {
		imageMeta, ok = GetAttachmentMetadata(c, attachmentID)
		if !ok {
			return ""
		}
	}
	return CalculateImageSrcset(c, src.Width, src.Height, src.URL, imageMeta, attachmentID)
}

// GetAttachmentImageSizes is wp_get_attachment_image_sizes (P8.45 / P8.46).
// P8.46: fires wp_get_attachment_image_sizes filter after calculate.
func GetAttachmentImageSizes(c *Context, attachmentID int64, size string, imageMeta map[string]any) string {
	src, ok := GetAttachmentImageSrc(c, attachmentID, size, false)
	if !ok {
		return ""
	}
	if imageMeta == nil {
		imageMeta, _ = GetAttachmentMetadata(c, attachmentID)
	}
	if size == "" {
		size = "medium"
	}
	sizes := CalculateImageSizes(c, src.Width, src.Height, src.URL, imageMeta, attachmentID)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_get_attachment_image_sizes", sizes, size, src.URL, imageMeta, attachmentID); v != nil {
			if s, ok := v.(string); ok {
				sizes = s
			}
		}
	}
	return sizes
}

// ImageSizeFromMeta mirrors _wp_get_image_size_from_meta (P8.46).
// Returns width/height for "full" or meta.sizes[name]; ok=false when missing.
func ImageSizeFromMeta(sizeName string, imageMeta map[string]any) (width, height int, ok bool) {
	if imageMeta == nil {
		return 0, 0, false
	}
	sizeName = strings.TrimSpace(sizeName)
	if sizeName == "" {
		return 0, 0, false
	}
	if sizeName == "full" {
		w := anyToInt(imageMeta["width"])
		h := anyToInt(imageMeta["height"])
		if w < 1 {
			return 0, 0, false
		}
		return w, h, true
	}
	entry := attachmentSizeEntry(imageMeta, sizeName)
	if entry == nil {
		return 0, 0, false
	}
	w := anyToInt(entry["width"])
	h := anyToInt(entry["height"])
	if w < 1 {
		return 0, 0, false
	}
	return w, h, true
}

// CalculateImageSizesFromName mirrors wp_calculate_image_sizes($size_name, ...) (P8.46).
// Resolves width from image meta then fires wp_calculate_image_sizes with the string size.
func CalculateImageSizesFromName(c *Context, sizeName, imageSrc string, imageMeta map[string]any, attachmentID int64) string {
	if imageMeta == nil && attachmentID > 0 {
		imageMeta, _ = GetAttachmentMetadata(c, attachmentID)
	}
	w, _, ok := ImageSizeFromMeta(sizeName, imageMeta)
	if !ok || w < 1 {
		return ""
	}
	sizes := fmt.Sprintf("(max-width: %dpx) 100vw, %dpx", w, w)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_calculate_image_sizes", sizes, sizeName, imageSrc, imageMeta, attachmentID); v != nil {
			if s, ok := v.(string); ok {
				sizes = s
			}
		}
	}
	return sizes
}

// CalculateImageSizes mirrors wp_calculate_image_sizes (P8.45 / P8.46).
// size filter arg is []int{width, height}; use CalculateImageSizesFromName for string sizes.
func CalculateImageSizes(c *Context, width, height int, imageSrc string, imageMeta map[string]any, attachmentID int64) string {
	_ = imageSrc
	_ = imageMeta
	if width < 1 {
		return ""
	}
	sizes := fmt.Sprintf("(max-width: %dpx) 100vw, %dpx", width, width)
	if c != nil && c.Hooks != nil {
		sizeArr := []int{width, height}
		if v := c.Hooks.ApplyFilters("wp_calculate_image_sizes", sizes, sizeArr, imageSrc, imageMeta, attachmentID); v != nil {
			if s, ok := v.(string); ok {
				sizes = s
			}
		}
	}
	return sizes
}

// CalculateImageSrcset mirrors wp_calculate_image_srcset (P8.45).
// Returns "" when unavailable (PHP false).
func CalculateImageSrcset(c *Context, width, height int, imageSrc string, imageMeta map[string]any, attachmentID int64) string {
	if imageMeta == nil {
		return ""
	}
	if c != nil && c.Hooks != nil {
		sizeArr := []int{width, height}
		if v := c.Hooks.ApplyFilters("wp_calculate_image_srcset_meta", imageMeta, sizeArr, imageSrc, attachmentID); v != nil {
			if m, ok := v.(map[string]any); ok {
				imageMeta = m
			}
		}
	}
	sizesRaw, hasSizes := imageMeta["sizes"]
	file := strings.TrimSpace(fmt.Sprint(imageMeta["file"]))
	if !hasSizes || file == "" || file == "<nil>" || len(file) < 4 {
		return ""
	}
	if width < 1 {
		return ""
	}

	imageSizes := map[string]map[string]any{}
	switch t := sizesRaw.(type) {
	case map[string]any:
		for k, v := range t {
			if m, ok := v.(map[string]any); ok {
				imageSizes[k] = m
			}
		}
	case map[any]any:
		if m, ok := phpValueToStringMap(t); ok {
			for k, v := range m {
				if sm, ok := v.(map[string]any); ok {
					imageSizes[k] = sm
				}
			}
		}
	}

	imageBasename := Basename(file, "")
	// Flattened GIF: if thumbnail mime is gif and src is full, skip srcset.
	if thumb, ok := imageSizes["thumbnail"]; ok {
		mime := strings.ToLower(strings.TrimSpace(fmt.Sprint(thumb["mime-type"])))
		if mime == "image/gif" {
			if strings.Contains(imageSrc, file) {
				return ""
			}
		} else {
			imageSizes["__full__"] = map[string]any{
				"width":  anyToInt(imageMeta["width"]),
				"height": anyToInt(imageMeta["height"]),
				"file":   imageBasename,
			}
		}
	} else {
		imageSizes["__full__"] = map[string]any{
			"width":  anyToInt(imageMeta["width"]),
			"height": anyToInt(imageMeta["height"]),
			"file":   imageBasename,
		}
	}

	dirname := attachmentUploadDirname(file)
	if dirname != "" && !strings.HasSuffix(dirname, "/") {
		dirname += "/"
	}
	uploads := GetUploadDir(c)
	imageBaseurl := strings.TrimRight(uploads.Baseurl, "/") + "/" + dirname

	editHash := ""
	imageEdited := false
	if m := reImageEditHash.FindString(Basename(imageSrc, "")); m != "" {
		imageEdited = true
		editHash = m
	}

	maxWidth := 2048
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("max_srcset_image_width", maxWidth, []int{width, height}); v != nil {
			switch t := v.(type) {
			case int:
				maxWidth = t
			case int64:
				maxWidth = int(t)
			case float64:
				maxWidth = int(t)
			}
		}
	}

	type source struct {
		url        string
		descriptor string
		value      int
	}
	sources := map[int]source{}
	srcMatched := false
	var srcWidth int

	for _, image := range imageSizes {
		if image == nil {
			continue
		}
		imgFile := strings.TrimSpace(fmt.Sprint(image["file"]))
		imgW := anyToInt(image["width"])
		imgH := anyToInt(image["height"])
		if imgFile == "" || imgFile == "<nil>" || imgW < 1 {
			continue
		}
		isSrc := false
		if !srcMatched && strings.Contains(imageSrc, dirname+imgFile) {
			srcMatched = true
			isSrc = true
			srcWidth = imgW
		}
		if imageEdited && !strings.Contains(imgFile, editHash) {
			continue
		}
		if maxWidth > 0 && imgW > maxWidth && !isSrc {
			continue
		}
		if !imageMatchesRatio(width, height, imgW, imgH) {
			continue
		}
		srcURL := imageBaseurl + imgFile
		srcURL = strings.ReplaceAll(srcURL, " ", "%20")
		sources[imgW] = source{url: srcURL, descriptor: "w", value: imgW}
	}

	if c != nil && c.Hooks != nil {
		// Convert to filter-friendly map[int]map
		filterSources := map[int]map[string]any{}
		for w, s := range sources {
			filterSources[w] = map[string]any{
				"url":        s.url,
				"descriptor": s.descriptor,
				"value":      s.value,
			}
		}
		if v := c.Hooks.ApplyFilters("wp_calculate_image_srcset", filterSources, []int{width, height}, imageSrc, imageMeta, attachmentID); v != nil {
			if m, ok := v.(map[int]map[string]any); ok {
				sources = map[int]source{}
				for w, sm := range m {
					sources[w] = source{
						url:        strings.TrimSpace(fmt.Sprint(sm["url"])),
						descriptor: strings.TrimSpace(fmt.Sprint(sm["descriptor"])),
						value:      anyToInt(sm["value"]),
					}
					if sources[w].value == 0 {
						sources[w] = source{url: sources[w].url, descriptor: sources[w].descriptor, value: w}
					}
				}
			}
		}
	}

	if !srcMatched || len(sources) < 2 {
		return ""
	}

	// Emit src candidate first (iOS8 quirk), then remaining by width.
	var widths []int
	for w := range sources {
		if w != srcWidth {
			widths = append(widths, w)
		}
	}
	sort.Ints(widths)
	if srcWidth > 0 {
		if _, ok := sources[srcWidth]; ok {
			widths = append([]int{srcWidth}, widths...)
		}
	}

	var parts []string
	seen := map[int]bool{}
	for _, w := range widths {
		if seen[w] {
			continue
		}
		seen[w] = true
		s := sources[w]
		if s.url == "" || s.value < 1 {
			continue
		}
		desc := s.descriptor
		if desc == "" {
			desc = "w"
		}
		parts = append(parts, s.url+" "+strconv.Itoa(s.value)+desc)
	}
	if len(parts) < 2 {
		return ""
	}
	return strings.Join(parts, ", ")
}

// imageMatchesRatio mirrors wp_image_matches_ratio (P8.45).
func imageMatchesRatio(sourceW, sourceH, targetW, targetH int) bool {
	if sourceW < 1 || sourceH < 1 || targetW < 1 || targetH < 1 {
		return false
	}
	var constrainedW, constrainedH int
	var expectedW, expectedH int
	if sourceW > targetW {
		constrainedW, constrainedH = constrainDimensionsForRatio(sourceW, sourceH, targetW, 0)
		expectedW, expectedH = targetW, targetH
	} else {
		constrainedW, constrainedH = constrainDimensionsForRatio(targetW, targetH, sourceW, 0)
		expectedW, expectedH = sourceW, sourceH
	}
	return fuzzyNumberMatch(constrainedW, expectedW) && fuzzyNumberMatch(constrainedH, expectedH)
}

func constrainDimensionsForRatio(currentW, currentH, maxW, maxH int) (int, int) {
	if maxW <= 0 && maxH <= 0 {
		return currentW, currentH
	}
	widthRatio := 1.0
	heightRatio := 1.0
	if maxW > 0 && currentW > 0 && currentW > maxW {
		widthRatio = float64(maxW) / float64(currentW)
	}
	if maxH > 0 && currentH > 0 && currentH > maxH {
		heightRatio = float64(maxH) / float64(currentH)
	}
	smaller := widthRatio
	if heightRatio < smaller {
		smaller = heightRatio
	}
	larger := widthRatio
	if heightRatio > larger {
		larger = heightRatio
	}
	ratio := larger
	overflowW := maxW > 0 && int(float64(currentW)*larger+0.5) > maxW
	overflowH := maxH > 0 && int(float64(currentH)*larger+0.5) > maxH
	if overflowW || overflowH {
		ratio = smaller
	}
	w := int(float64(currentW)*ratio + 0.5)
	h := int(float64(currentH)*ratio + 0.5)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// sizesAttributeIncludesAuto mirrors wp_sizes_attribute_includes_valid_auto (P8.45 / P8.46).
// True when the first comma-separated token is exactly "auto" (case-insensitive).
func sizesAttributeIncludesAuto(sizes string) bool {
	sizes = strings.TrimSpace(sizes)
	if sizes == "" {
		return false
	}
	first, _, _ := strings.Cut(sizes, ",")
	return strings.EqualFold(strings.TrimSpace(first), "auto")
}

// prependAutoSizes prefixes "auto, " when missing (P8.46 helper for lazy img tags).
func prependAutoSizes(sizes string) string {
	sizes = strings.TrimSpace(sizes)
	if sizes == "" || sizesAttributeIncludesAuto(sizes) {
		return sizes
	}
	return "auto, " + sizes
}

func fuzzyNumberMatch(a, b int) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= 1
}

// attachmentUploadDirname mirrors _wp_get_attachment_relative_path dirname (P8.45).
// Unlike attachmentRelativePath (P8.42), this returns only the directory segment.
func attachmentUploadDirname(file string) string {
	dirname := path.Dir(strings.ReplaceAll(file, "\\", "/"))
	if dirname == "." || dirname == "" {
		return ""
	}
	if idx := strings.Index(dirname, "wp-content/uploads"); idx >= 0 {
		dirname = strings.TrimPrefix(dirname[idx+len("wp-content/uploads"):], "/")
	}
	return dirname
}
