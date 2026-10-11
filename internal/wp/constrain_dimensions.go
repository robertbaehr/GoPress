package wp

import "math"

// ConstrainDimensions fits current width/height inside a max box while
// preserving aspect ratio (wp_constrain_dimensions, P8.60). When both max
// dimensions are 0, returns the current size. Applies wp_constrain_dimensions.
func ConstrainDimensions(c *Context, currentW, currentH, maxW, maxH int) (int, int) {
	if maxW == 0 && maxH == 0 {
		return applyConstrainDimensionsFilter(c, currentW, currentH, currentW, currentH, maxW, maxH)
	}
	widthRatio := 1.0
	heightRatio := 1.0
	didWidth := false
	didHeight := false
	if maxW > 0 && currentW > 0 && currentW > maxW {
		widthRatio = float64(maxW) / float64(currentW)
		didWidth = true
	}
	if maxH > 0 && currentH > 0 && currentH > maxH {
		heightRatio = float64(maxH) / float64(currentH)
		didHeight = true
	}
	smaller := math.Min(widthRatio, heightRatio)
	larger := math.Max(widthRatio, heightRatio)
	ratio := larger
	if int(math.Round(float64(currentW)*larger)) > maxW || int(math.Round(float64(currentH)*larger)) > maxH {
		ratio = smaller
	}
	w := int(math.Round(float64(currentW) * ratio))
	h := int(math.Round(float64(currentH) * ratio))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if didWidth && w == maxW-1 {
		w = maxW
	}
	if didHeight && h == maxH-1 {
		h = maxH
	}
	return applyConstrainDimensionsFilter(c, w, h, currentW, currentH, maxW, maxH)
}

func applyConstrainDimensionsFilter(c *Context, w, h, currentW, currentH, maxW, maxH int) (int, int) {
	if c == nil || c.Hooks == nil {
		return w, h
	}
	seed := []int{w, h}
	if v := c.Hooks.ApplyFilters("wp_constrain_dimensions", seed, currentW, currentH, maxW, maxH); v != nil {
		if dims, ok := v.([]int); ok && len(dims) >= 2 {
			return dims[0], dims[1]
		}
	}
	return w, h
}
