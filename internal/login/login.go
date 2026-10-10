// Package login implements wp-login.php (P6.1).
package login

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Handler serves GET/POST /wp-login.php.
type Handler struct{}

// Handle dispatches login form display and credential POST.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil {
		http.Error(w, "GoPress context unavailable\n", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.serveForm(w, r, c, "")
	case http.MethodPost:
		h.servePost(w, r, c)
	default:
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "Method Not Allowed\n", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) serveForm(w http.ResponseWriter, r *http.Request, c *wp.Context, errMsg string) {
	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo == "" {
		redirectTo = "/wp-admin/"
	}
	blogname := optionString(c, "blogname", "GoPress")

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Log In &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; GoPress</title>\n")
	b.WriteString("<link rel=\"stylesheet\" href=\"/wp-admin/css/login.min.css\">\n")
	b.WriteString("</head>\n<body class=\"login\">\n")
	b.WriteString("<div id=\"login\">\n")
	b.WriteString("<h1><a href=\"")
	b.WriteString(html.EscapeString(optionString(c, "home", "/")))
	b.WriteString("\">")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</a></h1>\n")
	if errMsg != "" {
		b.WriteString("<div id=\"login_error\"><strong>Error:</strong> ")
		b.WriteString(html.EscapeString(errMsg))
		b.WriteString("<br></div>\n")
	}
	b.WriteString("<form name=\"loginform\" id=\"loginform\" action=\"wp-login.php\" method=\"post\">\n")
	b.WriteString("<p>\n<label for=\"user_login\">Username or Email Address</label>\n")
	b.WriteString("<input type=\"text\" name=\"log\" id=\"user_login\" class=\"input\" value=\"\" size=\"20\" autocapitalize=\"off\" autocomplete=\"username\" required>\n</p>\n")
	b.WriteString("<p>\n<label for=\"user_pass\">Password</label>\n")
	b.WriteString("<input type=\"password\" name=\"pwd\" id=\"user_pass\" class=\"input\" value=\"\" size=\"20\" autocomplete=\"current-password\" required>\n</p>\n")
	b.WriteString("<p class=\"forgetmenot\"><label for=\"rememberme\"><input name=\"rememberme\" type=\"checkbox\" id=\"rememberme\" value=\"forever\"> Remember Me</label></p>\n")
	b.WriteString("<p class=\"submit\">\n")
	b.WriteString("<input type=\"submit\" name=\"wp-submit\" id=\"wp-submit\" class=\"button button-primary button-large\" value=\"Log In\">\n")
	b.WriteString("<input type=\"hidden\" name=\"redirect_to\" value=\"")
	b.WriteString(html.EscapeString(redirectTo))
	b.WriteString("\">\n")
	b.WriteString("<input type=\"hidden\" name=\"testcookie\" value=\"1\">\n")
	b.WriteString("</p>\n</form>\n</div>\n</body>\n</html>\n")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (h *Handler) servePost(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if err := r.ParseForm(); err != nil {
		h.serveForm(w, r, c, "Invalid form submission.")
		return
	}
	login := strings.TrimSpace(r.FormValue("log"))
	password := r.FormValue("pwd")
	remember := r.FormValue("rememberme") != ""
	redirectTo := r.FormValue("redirect_to")
	if redirectTo == "" {
		redirectTo = r.URL.Query().Get("redirect_to")
	}
	redirectTo = sanitizeRedirect(redirectTo)

	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	set, err := c.Login(login, password, remember, secure, 0)
	if err != nil || set == nil {
		// Re-display form with error; preserve redirect_to via query for GET-shaped form.
		q := r.URL.Query()
		q.Set("redirect_to", redirectTo)
		r.URL.RawQuery = q.Encode()
		h.serveForm(w, r, c, "The username or password you entered is incorrect.")
		return
	}
	set.WriteAuthCookies(w, "/")
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func sanitizeRedirect(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "/wp-admin/"
	}
	// Allow relative paths.
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "/wp-admin/"
	}
	// Absolute URLs only if same-ish path; for fixture simplicity accept any http(s) to local hosts.
	if u.Scheme != "http" && u.Scheme != "https" {
		return "/wp-admin/"
	}
	host := strings.ToLower(u.Hostname())
	if host == "127.0.0.1" || host == "localhost" || host == "" {
		if u.Path == "" {
			return "/"
		}
		return u.RequestURI()
	}
	return "/wp-admin/"
}

func optionString(c *wp.Context, name, fallback string) string {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case string:
		if t == "" {
			return fallback
		}
		return t
	default:
		s := fmt.Sprint(t)
		if s == "" || s == "<nil>" {
			return fallback
		}
		return s
	}
}
