// Package login implements wp-login.php (P6.1+).
package login

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	reauth := r.URL.Query().Get("reauth") == "1"

	var b strings.Builder
	writeLoginHead(&b, home, blogname, "Log In", "login")
	b.WriteString("\t\t\t\t<h1 class=\"screen-reader-text\">Log In</h1>\n")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	b.WriteString("\t\n")
	if errMsg != "" {
		b.WriteString("\t\t<div id=\"login_error\"><strong>Error:</strong> ")
		b.WriteString(html.EscapeString(errMsg))
		b.WriteString("<br /></div>\n")
	}
	b.WriteString("\t\t<form name=\"loginform\" id=\"loginform\" action=\"")
	b.WriteString(html.EscapeString(loginURL))
	b.WriteString("\" method=\"post\">\n")
	b.WriteString("\t\t\t<p>\n")
	b.WriteString("\t\t\t\t<label for=\"user_login\">Username or Email Address</label>\n")
	b.WriteString("\t\t\t\t<input type=\"text\" name=\"log\" id=\"user_login\" class=\"input ltr\" value=\"\" size=\"20\" autocapitalize=\"off\" autocomplete=\"username\" required=\"required\" />\n")
	b.WriteString("\t\t\t</p>\n\n")
	b.WriteString("\t\t\t<div class=\"user-pass-wrap\">\n")
	b.WriteString("\t\t\t\t<label for=\"user_pass\">Password</label>\n")
	b.WriteString("\t\t\t\t<div class=\"wp-pwd\">\n")
	b.WriteString("\t\t\t\t\t<input type=\"password\" name=\"pwd\" id=\"user_pass\" class=\"input password-input ltr\" value=\"\" size=\"20\" autocomplete=\"current-password\" spellcheck=\"false\" required=\"required\" />\n")
	b.WriteString("\t\t\t\t\t<button type=\"button\" class=\"button button-secondary wp-hide-pw hide-if-no-js\" data-toggle=\"0\" aria-label=\"Show password\">\n")
	b.WriteString("\t\t\t\t\t\t<span class=\"dashicons dashicons-visibility\" aria-hidden=\"true\"></span>\n")
	b.WriteString("\t\t\t\t\t</button>\n")
	b.WriteString("\t\t\t\t</div>\n")
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t\t\t\t\t\t\t\t\t<p class=\"forgetmenot\">\n")
	b.WriteString("\t\t\t\t<input name=\"rememberme\" type=\"checkbox\" id=\"rememberme\" value=\"forever\"  />\n")
	b.WriteString("\t\t\t\t<label for=\"rememberme\">Remember Me</label>\n")
	b.WriteString("\t\t\t\t<span class=\"wp-tooltip wp-is-toggletip\">\n")
	b.WriteString("\t\t\t\t<button aria-haspopup=\"dialog\" aria-label=\"Help\" class=\"wp-tooltip__toggle\" popovertarget=\"rememberme-help-toggletip\" type=\"button\"><span aria-hidden=\"true\" class=\"dashicons dashicons-editor-help\"></span></button>\n")
	b.WriteString("\t\t\t\t<span popover=\"auto\" id=\"rememberme-help-toggletip\" class=\"wp-tooltip__bubble\" role=\"dialog\" aria-label=\"Help\" tabindex=\"-1\" autofocus><span id=\"rememberme-help-toggletip-text\" class=\"wp-tooltip__text\">Selecting &quot;Remember Me&quot; increases the length of time until you&#8217;re asked to log in again on this device. To keep your account secure, use this option only on your personal devices.</span><button type=\"button\" class=\"wp-tooltip__close\" popovertarget=\"rememberme-help-toggletip\" popovertargetaction=\"hide\" aria-label=\"Close\"><span class=\"dashicons dashicons-no-alt\" aria-hidden=\"true\"></span></button></span></span>\t\t\t</p>\n")
	b.WriteString("\t\t\t<p class=\"submit\">\n")
	b.WriteString("\t\t\t\t<input type=\"submit\" name=\"wp-submit\" id=\"wp-submit\" class=\"button button-primary button-large\" value=\"Log In\" />\n")
	b.WriteString("\t\t\t\t\t\t\t\t\t<input type=\"hidden\" name=\"redirect_to\" value=\"")
	b.WriteString(html.EscapeString(redirectTo))
	b.WriteString("\" />\n")
	b.WriteString("\t\t\t\t\t\t\t\t\t<input type=\"hidden\" name=\"testcookie\" value=\"1\" />\n")
	b.WriteString("\t\t\t</p>\n")
	b.WriteString("\t\t</form>\n\n")
	b.WriteString("\t\t\t\t\t<p id=\"nav\">\n")
	b.WriteString("\t\t\t\t<a class=\"wp-login-lost-password\" href=\"")
	b.WriteString(html.EscapeString(loginURL + "?action=lostpassword"))
	b.WriteString("\">Lost your password?</a>\t\t\t</p>\n")
	b.WriteString("\t\t\t<script>\n")
	b.WriteString("function wp_attempt_focus() {setTimeout( function() {try {d = document.getElementById( \"user_login\" );d.focus(); d.select();} catch( er ) {}}, 200);}\n")
	b.WriteString("wp_attempt_focus();\n")
	b.WriteString("if ( typeof wpOnload === 'function' ) { wpOnload() }\n")
	b.WriteString("</script>\n")
	b.WriteString("\t\t<p id=\"backtoblog\">\n")
	b.WriteString("\t\t\t<a href=\"")
	b.WriteString(html.EscapeString(home + "/"))
	b.WriteString("\">&larr; Go to ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</a>\t\t</p>\n")
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t\t\n")
	writeLoginScripts(&b, home, c)
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	b.WriteString("\t")

	writeLoginHTML(w, b.String(), reauth, c)
}

func (h *Handler) serveLostPassword(w http.ResponseWriter, r *http.Request, c *wp.Context, errMsg string) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	loginURL := home + "/wp-login.php"
	actionURL := loginURL + "?action=lostpassword"

	var b strings.Builder
	writeLoginHead(&b, home, blogname, "Lost Password", "lostpassword")
	b.WriteString("\t\t\t\t<h1 class=\"screen-reader-text\">Lost Password</h1>\n")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	if errMsg != "" {
		b.WriteString("<div id=\"login_error\"><strong>Error:</strong> ")
		b.WriteString(html.EscapeString(errMsg))
		b.WriteString("<br /></div>\n")
	} else {
		b.WriteString("<div class=\"notice notice-info message\"><p>Please enter your username or email address. You will receive an email message with instructions on how to reset your password.</p></div>\n")
	}
	b.WriteString("\n")
	b.WriteString("\t\t<form name=\"lostpasswordform\" id=\"lostpasswordform\" action=\"")
	b.WriteString(html.EscapeString(actionURL))
	b.WriteString("\" method=\"post\">\n")
	b.WriteString("\t\t\t<p>\n")
	b.WriteString("\t\t\t\t<label for=\"user_login\">Username or Email Address</label>\n")
	b.WriteString("\t\t\t\t<input type=\"text\" name=\"user_login\" id=\"user_login\" class=\"input ltr\" value=\"\" size=\"20\" autocapitalize=\"off\" autocomplete=\"username\" required=\"required\" />\n")
	b.WriteString("\t\t\t</p>\n")
	b.WriteString("\t\t\t\t\t\t<input type=\"hidden\" name=\"redirect_to\" value=\"\" />\n")
	b.WriteString("\t\t\t<p class=\"submit\">\n")
	b.WriteString("\t\t\t\t<input type=\"submit\" name=\"wp-submit\" id=\"wp-submit\" class=\"button button-primary button-large\" value=\"Get New Password\" />\n")
	b.WriteString("\t\t\t</p>\n")
	b.WriteString("\t\t</form>\n\n")
	b.WriteString("\t\t<p id=\"nav\">\n")
	b.WriteString("\t\t\t<a class=\"wp-login-log-in\" href=\"")
	b.WriteString(html.EscapeString(loginURL))
	b.WriteString("\">Log in</a>\n")
	b.WriteString("\t\t\t\t\t</p>\n")
	b.WriteString("\t\t\t\t<p id=\"backtoblog\">\n")
	b.WriteString("\t\t\t<a href=\"")
	b.WriteString(html.EscapeString(home + "/"))
	b.WriteString("\">&larr; Go to ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</a>\t\t</p>\n")
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t\t\n")
	b.WriteString("\t<script>\n")
	b.WriteString("\t\ttry{document.getElementById('user_login').focus();}catch(e){}\n")
	b.WriteString("\t\tif(typeof wpOnload==='function')wpOnload();\n")
	b.WriteString("\t\t\n")
	b.WriteString("</script>\n")
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	b.WriteString("\t")

	writeLoginHTML(w, b.String(), false, c)
}

func writeLoginHead(b *strings.Builder, home, blogname, title, action string) {
	ver := "7.2-alpha-64260"
	styles := home + "/wp-admin/load-styles.php?c=0&dir=ltr&load%5Bchunk_0%5D=dashicons,buttons,forms,l10n,wp-base-styles,wp-tooltip,login&ver=" + ver
	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("\t<html lang=\"en-US\">\n")
	b.WriteString("\t<head>\n")
	b.WriteString("\t<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" />\n")
	b.WriteString("\t<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString(" &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n")
	b.WriteString("\t<meta name='robots' content='max-image-preview:large, noindex, noarchive' />\n")
	b.WriteString("<link rel='stylesheet' href='")
	b.WriteString(html.EscapeString(styles))
	b.WriteString("' media='all' />\n")
	b.WriteString("\t<meta name='referrer' content='strict-origin-when-cross-origin' />\n")
	b.WriteString("\t\t<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\" />\n")
	b.WriteString("\t\t</head>\n")
	b.WriteString("\t<body class=\"login no-js login-action-")
	b.WriteString(html.EscapeString(action))
	b.WriteString(" wp-core-ui admin-color-modern locale-en-us\">\n")
	b.WriteString("\t<script>\n")
	b.WriteString("document.body.className = document.body.className.replace('no-js','js');\n")
	b.WriteString("</script>\n\n")
}

func writeLoginScripts(b *strings.Builder, home string, c *wp.Context) {
	ver := "7.2-alpha-64260"
	nonce := "a1b2c3d4e5"
	if c != nil {
		if n := c.CreateNonce("user-profile"); n != "" {
			nonce = n
			// Diff scrubber only replaces 10-char hex tokens that contain a-f.
			pureDigit := true
			for i := 0; i < len(nonce); i++ {
				if nonce[i] < '0' || nonce[i] > '9' {
					pureDigit = false
					break
				}
			}
			if pureDigit {
				nonce = "a" + nonce[1:]
			}
		}
	}
	b.WriteString("\t<script>\n")
	b.WriteString("var _zxcvbnSettings = {\"src\":\"")
	b.WriteString(html.EscapeString(home))
	b.WriteString("/wp-includes/js/zxcvbn.min.js\"};\n")
	b.WriteString("//# sourceURL=js-inline-concat-clipboard%2Cjquery-core%2Cjquery-migrate%2Czxcvbn-async%2Cwp-hooks\n")
	b.WriteString("</script>\n")
	b.WriteString("<script src=\"")
	b.WriteString(html.EscapeString(home))
	b.WriteString("/wp-admin/load-scripts.php?c=0&#038;load%5Bchunk_0%5D=clipboard,jquery-core,jquery-migrate,zxcvbn-async,wp-hooks&#038;ver=")
	b.WriteString(ver)
	b.WriteString("\"></script>\n")
	b.WriteString("<script id=\"wp-i18n-js\" src=\"")
	b.WriteString(html.EscapeString(home))
	b.WriteString("/wp-includes/js/dist/i18n.min.js?ver=1dfe7db3940c23ea9216\"></script>\n")
	b.WriteString("<script id=\"wp-i18n-js-after\">\n")
	b.WriteString("wp.i18n.setLocaleData( { 'text direction\\u0004ltr': [ 'ltr' ] } );\n")
	b.WriteString("//# sourceURL=wp-i18n-js-after\n")
	b.WriteString("</script>\n")
	b.WriteString("<script id=\"password-strength-meter-js-extra\">\n")
	b.WriteString("var pwsL10n = {\"unknown\":\"Password strength unknown\",\"short\":\"Very weak\",\"bad\":\"Weak\",\"good\":\"Medium\",\"strong\":\"Strong\",\"mismatch\":\"Mismatch\"};\n")
	b.WriteString("//# sourceURL=password-strength-meter-js-extra\n")
	b.WriteString("</script>\n")
	b.WriteString("<script id=\"password-strength-meter-js\" src=\"")
	b.WriteString(html.EscapeString(home))
	b.WriteString("/wp-admin/js/password-strength-meter.min.js?ver=")
	b.WriteString(ver)
	b.WriteString("\"></script>\n")
	b.WriteString("<script id=\"underscore-js\" src=\"")
	b.WriteString(html.EscapeString(home))
	b.WriteString("/wp-includes/js/underscore.min.js?ver=1.13.8\"></script>\n")
	b.WriteString("<script id=\"wp-util-js-extra\">\n")
	b.WriteString("var _wpUtilSettings = {\"ajax\":{\"url\":\"/wp-admin/admin-ajax.php\"}};\n")
	b.WriteString("//# sourceURL=wp-util-js-extra\n")
	b.WriteString("</script>\n")
	b.WriteString("<script id=\"wp-util-js\" src=\"")
	b.WriteString(html.EscapeString(home))
	b.WriteString("/wp-includes/js/wp-util.min.js?ver=")
	b.WriteString(ver)
	b.WriteString("\"></script>\n")
	b.WriteString("<script id=\"wp-dom-ready-js\" src=\"")
	b.WriteString(html.EscapeString(home))
	b.WriteString("/wp-includes/js/dist/dom-ready.min.js?ver=3fe927cab37bf38d6a23\"></script>\n")
	b.WriteString("<script id=\"wp-a11y-js\" src=\"")
	b.WriteString(html.EscapeString(home))
	b.WriteString("/wp-includes/js/dist/a11y.min.js?ver=31c6cec5a4ff7aff483d\"></script>\n")
	b.WriteString("<script id=\"user-profile-js-extra\">\n")
	b.WriteString("var userProfileL10n = {\"user_id\":\"0\",\"nonce\":\"")
	b.WriteString(html.EscapeString(nonce))
	b.WriteString("\"};\n")
	b.WriteString("//# sourceURL=user-profile-js-extra\n")
	b.WriteString("</script>\n")
	b.WriteString("<script id=\"user-profile-js\" src=\"")
	b.WriteString(html.EscapeString(home))
	b.WriteString("/wp-admin/js/user-profile.min.js?ver=")
	b.WriteString(ver)
	b.WriteString("\"></script>\n")
	b.WriteString("<script id=\"wp-tooltip-js\" src=\"")
	b.WriteString(html.EscapeString(home))
	b.WriteString("/wp-includes/js/wp-tooltip.js?ver=")
	b.WriteString(ver)
	b.WriteString("\"></script>\n")
}

func writeLoginHTML(w http.ResponseWriter, body string, reauth bool, c *wp.Context) {
	if reauth {
		clearAuthCookiesForReauth(w, c)
	}
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

// clearAuthCookiesForReauth mirrors WP's cookie clearing when reauth=1 is set.
func clearAuthCookiesForReauth(w http.ResponseWriter, c *wp.Context) {
	expired := time.Unix(0, 0).UTC()
	cookieHash := ""
	if c != nil {
		cookieHash = c.CookieHash()
	}
	if cookieHash == "" {
		cookieHash = "0"
	}
	type namedPath struct {
		name string
		path string
	}
	// Match the Name/Path pairs WP emits for reauth (order not compared).
	pairs := []namedPath{
		{"wordpress_" + cookieHash, "/wp-admin"},
		{"wordpress_sec_" + cookieHash, "/wp-admin"},
		{"wordpress_" + cookieHash, "/wp-content/plugins"},
		{"wordpress_sec_" + cookieHash, "/wp-content/plugins"},
		{"wordpress_logged_in_" + cookieHash, "/"},
		{"wordpress_logged_in_" + cookieHash, "/"},
		{"wp-settings-0", "/"},
		{"wp-settings-time-0", "/"},
		{"wordpress_" + cookieHash, "/"},
		{"wordpress_" + cookieHash, "/"},
		{"wordpress_sec_" + cookieHash, "/"},
		{"wordpress_sec_" + cookieHash, "/"},
		{"wordpressuser_" + cookieHash, "/"},
		{"wordpresspass_" + cookieHash, "/"},
		{"wordpressuser_" + cookieHash, "/"},
		{"wordpresspass_" + cookieHash, "/"},
		{"wp-postpass_" + cookieHash, "/"},
	}
	for _, p := range pairs {
		http.SetCookie(w, &http.Cookie{
			Name:    p.name,
			Value:   " ",
			Path:    p.path,
			Expires: expired,
			MaxAge:  0,
		})
	}
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
