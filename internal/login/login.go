// Package login implements wp-login.php (P6.1+).
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
	action := r.URL.Query().Get("action")
	if action == "" && r.Method == http.MethodPost {
		_ = r.ParseForm()
		action = r.FormValue("action")
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		switch action {
		case "lostpassword", "retrievepassword":
			h.serveLostPassword(w, r, c, "")
		case "logout":
			h.serveLogout(w, r, c)
		default:
			h.serveForm(w, r, c, "")
		}
	case http.MethodPost:
		switch action {
		case "lostpassword", "retrievepassword":
			// Stub: redisplay form (mail reset lands later).
			h.serveLostPassword(w, r, c, "")
		case "logout":
			h.serveLogout(w, r, c)
		default:
			h.servePost(w, r, c)
		}
	default:
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "Method Not Allowed\n", http.StatusMethodNotAllowed)
	}
}

// serveLogout clears auth cookies and redirects (wp-login.php?action=logout, P6.6).
func (h *Handler) serveLogout(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	c.AuthenticateRequest()
	if c.CurrentUser != nil && c.SessionToken != "" {
		_ = c.DestroySessionToken(c.CurrentUser.ID, c.SessionToken)
	}
	c.ClearAuthCookies(w, "/")
	c.CurrentUser = nil
	c.SessionToken = ""

	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo == "" {
		_ = r.ParseForm()
		redirectTo = r.FormValue("redirect_to")
	}
	redirectTo = sanitizeRedirect(redirectTo)
	if redirectTo == "/wp-admin/" || strings.HasPrefix(redirectTo, "/wp-admin") {
		home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		redirectTo = home + "/wp-login.php?loggedout=true"
		if home == "" {
			redirectTo = "/wp-login.php?loggedout=true"
		}
	}
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func (h *Handler) serveForm(w http.ResponseWriter, r *http.Request, c *wp.Context, errMsg string) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo == "" {
		redirectTo = home + "/wp-admin/"
	}
	loginURL := home + "/wp-login.php"

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("<html lang=\"en-US\">\n")
	b.WriteString("<head>\n")
	b.WriteString("<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" />\n")
	b.WriteString("<title>Log In &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\" />\n")
	b.WriteString("<link rel=\"stylesheet\" href=\"/wp-admin/css/login.min.css\">\n")
	b.WriteString("</head>\n")
	b.WriteString("<body class=\"login no-js login-action-login wp-core-ui\">\n")
	b.WriteString("<script>document.body.className = document.body.className.replace('no-js','js');</script>\n")
	b.WriteString("<h1 class=\"screen-reader-text\">Log In</h1>\n")
	b.WriteString("<div id=\"login\">\n")
	b.WriteString("<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	if errMsg != "" {
		b.WriteString("<div id=\"login_error\"><strong>Error:</strong> ")
		b.WriteString(html.EscapeString(errMsg))
		b.WriteString("<br></div>\n")
	}
	b.WriteString("<form name=\"loginform\" id=\"loginform\" action=\"")
	b.WriteString(html.EscapeString(loginURL))
	b.WriteString("\" method=\"post\">\n")
	b.WriteString("<p>\n<label for=\"user_login\">Username or Email Address</label>\n")
	b.WriteString("<input type=\"text\" name=\"log\" id=\"user_login\" class=\"input ltr\" value=\"\" size=\"20\" autocapitalize=\"off\" autocomplete=\"username\" required=\"required\" />\n</p>\n")
	b.WriteString("<div class=\"user-pass-wrap\">\n")
	b.WriteString("<label for=\"user_pass\">Password</label>\n")
	b.WriteString("<div class=\"wp-pwd\">\n")
	b.WriteString("<input type=\"password\" name=\"pwd\" id=\"user_pass\" class=\"input password-input ltr\" value=\"\" size=\"20\" autocomplete=\"current-password\" spellcheck=\"false\" required=\"required\" />\n")
	b.WriteString("</div>\n</div>\n")
	b.WriteString("<p class=\"forgetmenot\">\n")
	b.WriteString("<input name=\"rememberme\" type=\"checkbox\" id=\"rememberme\" value=\"forever\" />\n")
	b.WriteString("<label for=\"rememberme\">Remember Me</label>\n</p>\n")
	b.WriteString("<p class=\"submit\">\n")
	b.WriteString("<input type=\"submit\" name=\"wp-submit\" id=\"wp-submit\" class=\"button button-primary button-large\" value=\"Log In\" />\n")
	b.WriteString("<input type=\"hidden\" name=\"redirect_to\" value=\"")
	b.WriteString(html.EscapeString(redirectTo))
	b.WriteString("\" />\n")
	b.WriteString("<input type=\"hidden\" name=\"testcookie\" value=\"1\" />\n")
	b.WriteString("</p>\n</form>\n")
	b.WriteString("<p id=\"nav\">\n")
	b.WriteString("<a class=\"wp-login-lost-password\" href=\"")
	b.WriteString(html.EscapeString(loginURL + "?action=lostpassword"))
	b.WriteString("\">Lost your password?</a>\n</p>\n")
	b.WriteString("<p id=\"backtoblog\">\n<a href=\"")
	b.WriteString(html.EscapeString(home + "/"))
	b.WriteString("\">&larr; Go to ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</a></p>\n")
	b.WriteString("</div>\n</body>\n</html>\n")

	writeLoginHTML(w, b.String())
}

func (h *Handler) serveLostPassword(w http.ResponseWriter, r *http.Request, c *wp.Context, errMsg string) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	loginURL := home + "/wp-login.php"
	actionURL := loginURL + "?action=lostpassword"

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("<html lang=\"en-US\">\n")
	b.WriteString("<head>\n")
	b.WriteString("<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" />\n")
	b.WriteString("<title>Lost Password &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\" />\n")
	b.WriteString("<link rel=\"stylesheet\" href=\"/wp-admin/css/login.min.css\">\n")
	b.WriteString("</head>\n")
	b.WriteString("<body class=\"login no-js login-action-lostpassword wp-core-ui\">\n")
	b.WriteString("<script>document.body.className = document.body.className.replace('no-js','js');</script>\n")
	b.WriteString("<h1 class=\"screen-reader-text\">Lost Password</h1>\n")
	b.WriteString("<div id=\"login\">\n")
	b.WriteString("<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	if errMsg != "" {
		b.WriteString("<div id=\"login_error\"><strong>Error:</strong> ")
		b.WriteString(html.EscapeString(errMsg))
		b.WriteString("<br></div>\n")
	} else {
		b.WriteString("<div class=\"notice notice-info message\"><p>Please enter your username or email address. You will receive an email message with instructions on how to reset your password.</p></div>\n")
	}
	b.WriteString("<form name=\"lostpasswordform\" id=\"lostpasswordform\" action=\"")
	b.WriteString(html.EscapeString(actionURL))
	b.WriteString("\" method=\"post\">\n")
	b.WriteString("<p>\n<label for=\"user_login\">Username or Email Address</label>\n")
	b.WriteString("<input type=\"text\" name=\"user_login\" id=\"user_login\" class=\"input ltr\" value=\"\" size=\"20\" autocapitalize=\"off\" autocomplete=\"username\" required=\"required\" />\n</p>\n")
	b.WriteString("<input type=\"hidden\" name=\"redirect_to\" value=\"\" />\n")
	b.WriteString("<p class=\"submit\">\n")
	b.WriteString("<input type=\"submit\" name=\"wp-submit\" id=\"wp-submit\" class=\"button button-primary button-large\" value=\"Get New Password\" />\n")
	b.WriteString("</p>\n</form>\n")
	b.WriteString("<p id=\"nav\">\n")
	b.WriteString("<a class=\"wp-login-log-in\" href=\"")
	b.WriteString(html.EscapeString(loginURL))
	b.WriteString("\">Log in</a>\n</p>\n")
	b.WriteString("<p id=\"backtoblog\">\n<a href=\"")
	b.WriteString(html.EscapeString(home + "/"))
	b.WriteString("\">&larr; Go to ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</a></p>\n")
	b.WriteString("</div>\n</body>\n</html>\n")

	writeLoginHTML(w, b.String())
}

func writeLoginHTML(w http.ResponseWriter, body string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "wordpress_test_cookie",
		Value:    "WP Cookie check",
		Path:     "/",
		HttpOnly: true,
	})
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
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
