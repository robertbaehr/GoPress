package wp

// WlwmanifestLink is wlwmanifest_link (P3.134).
// Core removed the Windows Live Writer head link in WP 6.3; this is a no-op
// for API parity so callers and front chrome never emit the link.
func WlwmanifestLink() string {
	return ""
}
