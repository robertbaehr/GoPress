package server

import (
	"fmt"
	"net/http"
)

// serveInstallPlaceholder renders a minimal WordPress install stub page (P6.4).
func serveInstallPlaceholder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate, max-age=0")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>WordPress &rsaquo; Installation</title>
</head>
<body class="wp-core-ui">
<h1>Welcome</h1>
<p>WordPress is not installed yet.</p>
<p>GoPress install stub (P6.4) — full installer lands later.</p>
<p><a href="/wp-admin/install.php">Continue to installation</a></p>
</body>
</html>
`)
	_ = r
}
