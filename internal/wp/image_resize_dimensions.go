package wp

import "math"

// ImageResizeDimensionsResult matches imagecopyresampled params from
// image_resize_dimensions (P8.61): dstX, dstY, srcX, srcY, dstW, dstH, srcW, srcH.
type ImageResizeDimensionsResult struct {
	DstX, DstY, SrcX, SrcY, DstW, DstH, SrcW, SrcH int
}

// ImageResizeDimensions calculates resize/crop source and dest rectangles
// (image_resize_dimensions, P8.61). Returns ok=false when inputs are invalid
// or would upscale. crop=true uses centered crop; crop=false scales to fit.
// Applies image_resize_dimensions pre-filter (non-nil short-circuit).
func ImageResizeDimensions(c *Context, origW, origH, destW, destH int, crop bool) (ImageResizeDimensionsResult, bool) {
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("image_resize_dimensions", nil, origW, origH, destW, destH, crop); v != nil {
			if r, ok := v.(ImageResizeDimensionsResult); ok {
				return r, true
			}
			if !okBool(v) {
				return ImageResizeDimensionsResult{}, false
			}
		}
	}
	if origW <= 0 || origH <= 0 {
		return ImageResizeDimensionsResult{}, false
	}
	if destW <= 0 && destH <= 0 {
		return ImageResizeDimensionsResult{}, false
	}
	// Stop if destination is larger than original (no upscale).
	if destH <= 0 {
		if origW < destW {
			return ImageResizeDimensionsResult{}, false
		}
	} else if destW <= 0 {
		if origH < destH {
			return ImageResizeDimensionsResult{}, false
		}
	} else if origW < destW && origH < destH {
		return ImageResizeDimensionsResult{}, false
	}

	if crop {
		aspect := float64(origW) / float64(origH)
		newW := destW
		newH := destH
		if newW <= 0 || newW > origW {
			newW = origW
		}
		if newH <= 0 || newH > origH {
			newH = origH
		}
		if newW == 0 {
			newW = int(math.Round(float64(newH) * aspect))
		}
		if newH == 0 {
			newH = int(math.Round(float64(newW) / aspect))
		}
		sizeRatio := math.Max(float64(newW)/float64(origW), float64(newH)/float64(origH))
		cropW := int(math.Round(float64(newW) / sizeRatio))
		cropH := int(math.Round(float64(newH) / sizeRatio))
		sX := (origW - cropW) / 2
		sY := (origH - cropH) / 2
		if sX < 0 {
			sX = 0
		}
		if sY < 0 {
			sY = 0
		}
		return ImageResizeDimensionsResult{0, 0, sX, sY, newW, newH, cropW, cropH}, true
	}

	// Scale to fit within dest box.
	ratioW := 1.0
	ratioH := 1.0
	if destW > 0 {
		ratioW = float64(destW) / float64(origW)
	}
	if destH > 0 {
		ratioH = float64(destH) / float64(origH)
	}
	ratio := math.Min(ratioW, ratioH)
	if destW <= 0 {
		ratio = ratioH
	}
	if destH <= 0 {
		ratio = ratioW
	}
	newW := int(math.Round(float64(origW) * ratio))
	newH := int(math.Round(float64(origH) * ratio))
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	return ImageResizeDimensionsResult{0, 0, 0, 0, newW, newH, origW, origH}, true
}

func okBool(v any) bool {
	b, ok := v.(bool)
	return ok && b
}
