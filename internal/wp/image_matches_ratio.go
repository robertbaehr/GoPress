package wp

// ImageMatchesRatio reports whether two dimension pairs share the same aspect
// ratio within a one-pixel fuzz (wp_image_matches_ratio, P8.62).
func ImageMatchesRatio(sourceW, sourceH, targetW, targetH int) bool {
	return imageMatchesRatio(sourceW, sourceH, targetW, targetH)
}
