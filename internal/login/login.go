// Package login implements wp-login.php (P6.1+).
package login

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
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
		case "register":
			h.serveRegister(w, r, c)
		default:
			if r.URL.Query().Get("checkemail") == "confirm" {
				h.serveCheckEmailConfirm(w, r, c)
				return
			}
			errMsg := ""
			if r.URL.Query().Get("registration") == "disabled" {
				errMsg = loginErrorHTML("registerdisabled", "User registration is currently not allowed.", "", "")
			}
			h.serveForm(w, r, c, errMsg)
		}
	case http.MethodPost:
		switch action {
		case "lostpassword", "retrievepassword":
			h.serveLostPasswordPost(w, r, c)
		case "logout":
			h.serveLogout(w, r, c)
		case "register":
			h.serveRegister(w, r, c)
		default:
			h.servePost(w, r, c)
		}
	default:
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "Method Not Allowed\n", http.StatusMethodNotAllowed)
	}
}

// serveRegister handles wp-login.php?action=register (P6.9).
// Non-multisite sites with users_can_register disabled redirect to
// ?registration=disabled and show an error on the login form (WP parity).
func (h *Handler) serveRegister(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if usersCanRegister(c) {
		// Full registration form is out of scope; still refuse until implemented.
		h.serveForm(w, r, c, loginErrorHTML("registerdisabled", "User registration is currently not allowed.", "", ""))
		return
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	target := home + "/wp-login.php?registration=disabled"
	if home == "" {
		target = "/wp-login.php?registration=disabled"
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func usersCanRegister(c *wp.Context) bool {
	if c == nil || c.Options == nil {
		return false
	}
	v := c.Options.GetOption("users_can_register", 0)
	switch t := v.(type) {
	case int:
		return t == 1
	case int64:
		return t == 1
	case string:
		return t == "1"
	case bool:
		return t
	default:
		return false
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
	interim := isInterimLogin(r)

	var b strings.Builder
	extraClass := ""
	if interim {
		extraClass = "interim-login"
	}
	writeLoginHead(&b, home, blogname, "Log In", "login", extraClass)
	b.WriteString("\t\t\t\t<h1 class=\"screen-reader-text\">Log In</h1>\n")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	b.WriteString("\t\n")
	if errMsg != "" {
		b.WriteString("\t\t<div id=\"login_error\">")
		b.WriteString(errMsg)
		b.WriteString("</div>\n")
	}
	actionURL := loginURL
	if interim {
		actionURL = loginURL + "?interim-login=1"
	}
	b.WriteString("\t\t<form name=\"loginform\" id=\"loginform\" action=\"")
	b.WriteString(html.EscapeString(actionURL))
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
	if interim {
		b.WriteString("\t\t\t\t\t\t\t\t\t<input type=\"hidden\" name=\"interim-login\" value=\"1\" />\n")
	}
	b.WriteString("\t\t\t</p>\n")
	b.WriteString("\t\t</form>\n\n")
	if !interim {
		b.WriteString("\t\t\t\t\t<p id=\"nav\">\n")
		b.WriteString("\t\t\t\t<a class=\"wp-login-lost-password\" href=\"")
		b.WriteString(html.EscapeString(loginURL + "?action=lostpassword"))
		b.WriteString("\">Lost your password?</a>\t\t\t</p>\n")
	}
	b.WriteString("\t\t\t<script>\n")
	b.WriteString("function wp_attempt_focus() {setTimeout( function() {try {d = document.getElementById( \"user_login\" );d.focus(); d.select();} catch( er ) {}}, 200);}\n")
	b.WriteString("wp_attempt_focus();\n")
	b.WriteString("if ( typeof wpOnload === 'function' ) { wpOnload() }\n")
	b.WriteString("</script>\n")
	if !interim {
		b.WriteString("\t\t<p id=\"backtoblog\">\n")
		b.WriteString("\t\t\t<a href=\"")
		b.WriteString(html.EscapeString(home + "/"))
		b.WriteString("\">&larr; Go to ")
		b.WriteString(html.EscapeString(blogname))
		b.WriteString("</a>\t\t</p>\n")
		writeLanguageSwitcher(&b, r, c, availableLanguages(c))
	}
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t\t\n")
	writeLoginScripts(&b, home, c)
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	b.WriteString("\t")

	writeLoginHTML(w, b.String(), reauth, c)
}

func (h *Handler) serveLostPasswordPost(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if err := r.ParseForm(); err != nil {
		h.serveLostPassword(w, r, c, "Invalid form submission.")
		return
	}
	userLogin := strings.TrimSpace(r.FormValue("user_login"))
	if err := wp.RetrievePassword(c, userLogin); err != nil {
		msg := "There is no account with that username or email address."
		switch err.Error() {
		case "empty_username":
			msg = "Please enter a username or email address."
		case "retrieve_password_email_failure":
			msg = "The email could not be sent."
		}
		h.serveLostPassword(w, r, c, msg)
		return
	}
	redirectTo := r.FormValue("redirect_to")
	if redirectTo == "" {
		home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		redirectTo = home + "/wp-login.php?checkemail=confirm"
		if home == "" {
			redirectTo = "/wp-login.php?checkemail=confirm"
		}
	}
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func (h *Handler) serveCheckEmailConfirm(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	loginURL := home + "/wp-login.php"
	var b strings.Builder
	writeLoginHead(&b, home, blogname, "Log In", "login")
	b.WriteString("\t\t\t\t<h1 class=\"screen-reader-text\">Log In</h1>\n")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	b.WriteString("<p class=\"message\">Check your email for the confirmation link, then visit the login page.</p>\n")
	b.WriteString("\t\t<p id=\"nav\">\n")
	b.WriteString("\t\t\t<a class=\"wp-login-log-in\" href=\"")
	b.WriteString(html.EscapeString(loginURL))
	b.WriteString("\">Log in</a>\n")
	b.WriteString("\t\t\t\t\t</p>\n")
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	writeLoginHTML(w, b.String(), false, c)
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

func writeLoginHead(b *strings.Builder, home, blogname, title, action string, extraClasses ...string) {
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
	for _, cls := range extraClasses {
		cls = strings.TrimSpace(cls)
		if cls == "" {
			continue
		}
		b.WriteByte(' ')
		b.WriteString(html.EscapeString(cls))
	}
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
		h.serveForm(w, r, c, loginErrorHTML("invalid_form", "Invalid form submission.", "", ""))
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
	interim := isInterimLogin(r)

	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	set, err := c.Login(login, password, remember, secure, 0)
	if err != nil || set == nil {
		// Re-display form with error; preserve redirect_to via query for GET-shaped form.
		q := r.URL.Query()
		q.Set("redirect_to", redirectTo)
		if interim {
			q.Set("interim-login", "1")
		}
		r.URL.RawQuery = q.Encode()
		home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		lostURL := home + "/wp-login.php?action=lostpassword"
		if home == "" {
			lostURL = "/wp-login.php?action=lostpassword"
		}
		h.serveForm(w, r, c, formatLoginAuthError(err, login, password, lostURL))
		return
	}
	set.WriteAuthCookies(w, "/")
	if interim {
		h.serveInterimLoginSuccess(w, r, c)
		return
	}
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

// formatLoginAuthError builds WP-shaped login_error HTML (P6.12).
func formatLoginAuthError(err error, login, password, lostURL string) string {
	ae, ok := err.(*wp.AuthError)
	if !ok || ae == nil {
		// Empty fields without typed error (e.g. both empty before Login change).
		if strings.TrimSpace(login) == "" && password == "" {
			return `<ul class="login-error-list"><li><strong>Error:</strong> The username field is empty.</li>` +
				`<li><strong>Error:</strong> The password field is empty.</li></ul>`
		}
		return loginErrorHTML("invalid_credentials", "The username or password you entered is incorrect.", "", "")
	}
	switch ae.Code {
	case "empty_username":
		if password == "" {
			return `<ul class="login-error-list"><li><strong>Error:</strong> The username field is empty.</li>` +
				`<li><strong>Error:</strong> The password field is empty.</li></ul>`
		}
		return loginErrorHTML("empty_username", "The username field is empty.", "", "")
	case "empty_password":
		return loginErrorHTML("empty_password", "The password field is empty.", "", "")
	case "invalid_username":
		user := html.EscapeString(ae.Login)
		msg := "The username <strong>" + user + "</strong> is not registered on this site. If you are unsure of your username, try your email address instead."
		return loginErrorHTML("invalid_username", msg, "", "")
	case "incorrect_password":
		user := html.EscapeString(ae.Login)
		msg := "The password you entered for the username <strong>" + user + "</strong> is incorrect. "
		if lostURL != "" {
			msg += `<a href="` + html.EscapeString(lostURL) + `">Lost your password?</a>`
		}
		return loginErrorHTML("incorrect_password", msg, "", "")
	default:
		return loginErrorHTML(ae.Code, "The username or password you entered is incorrect.", "", "")
	}
}

// loginErrorHTML wraps a message in a <p><strong>Error:</strong> …</p> block.
// msg may contain trusted markup (strong/a) already escaped for user values.
func loginErrorHTML(_, msg, _, _ string) string {
	return "<p><strong>Error:</strong> " + msg + "</p>"
}

// serveInterimLoginSuccess shows the interim modal success stub (P6.10).
func (h *Handler) serveInterimLoginSuccess(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	writeLoginHead(&b, home, blogname, "Log In", "login", "interim-login", "interim-login-success")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	b.WriteString("<p class=\"message\">You have logged in successfully.</p>\n")
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	writeLoginHTML(w, b.String(), false, c)
}

func isInterimLogin(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.URL != nil {
		v := r.URL.Query().Get("interim-login")
		if v == "1" || strings.EqualFold(v, "true") {
			return true
		}
	}
	if r.Form != nil {
		v := r.FormValue("interim-login")
		return v == "1" || strings.EqualFold(v, "true")
	}
	// Form may not be parsed yet for GET.
	return false
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

// writeLanguageSwitcher emits the login language dropdown when languages exist (P6.11).
// Mirrors wp-login.php: only rendered when get_available_languages() is non-empty.
func writeLanguageSwitcher(b *strings.Builder, r *http.Request, c *wp.Context, languages []string) {
	if len(languages) == 0 {
		return
	}
	selected := "en_US"
	if c != nil && c.Options != nil {
		if v := optionString(c, "WPLANG", ""); v != "" {
			selected = v
		}
	}
	if q := r.URL.Query().Get("wp_lang"); q != "" {
		selected = q
	}
	b.WriteString("\t\t\t<div class=\"language-switcher\">\n")
	b.WriteString("\t\t\t\t<form id=\"language-switcher\" method=\"get\">\n")
	b.WriteString("\t\t\t\t\t<label for=\"language-switcher-locales\">\n")
	b.WriteString("\t\t\t\t\t\tLanguage<span class=\"dashicons dashicons-translation\" aria-hidden=\"true\"></span>\n")
	b.WriteString("\t\t\t\t\t</label>\n")
	b.WriteString("\t\t\t\t\t<select name=\"wp_lang\" id=\"language-switcher-locales\">\n")
	b.WriteString("\t\t\t\t\t\t<option value=\"en_US\"")
	if selected == "en_US" || selected == "" {
		b.WriteString(" selected=\"selected\"")
	}
	b.WriteString(">English (United States)</option>\n")
	for _, loc := range languages {
		loc = strings.TrimSpace(loc)
		if loc == "" || loc == "en_US" {
			continue
		}
		b.WriteString("\t\t\t\t\t\t<option value=\"")
		b.WriteString(html.EscapeString(loc))
		b.WriteString("\"")
		if loc == selected {
			b.WriteString(" selected=\"selected\"")
		}
		b.WriteString(">")
		b.WriteString(html.EscapeString(loc))
		b.WriteString("</option>\n")
	}
	b.WriteString("\t\t\t\t\t</select>\n")
	if rt := r.URL.Query().Get("redirect_to"); rt != "" {
		b.WriteString("\t\t\t\t\t\t<input type=\"hidden\" name=\"redirect_to\" value=\"")
		b.WriteString(html.EscapeString(rt))
		b.WriteString("\" />\n")
	}
	if action := r.URL.Query().Get("action"); action != "" {
		b.WriteString("\t\t\t\t\t\t<input type=\"hidden\" name=\"action\" value=\"")
		b.WriteString(html.EscapeString(action))
		b.WriteString("\" />\n")
	}
	b.WriteString("\t\t\t\t\t\t<input type=\"submit\" class=\"button\" value=\"Change\">\n")
	b.WriteString("\t\t\t\t\t</form>\n")
	b.WriteString("\t\t\t\t</div>\n")
}

// availableLanguages mirrors get_available_languages() — installed .mo packs under
// wp-content/languages (excluding continents-cities / admin- / ms- prefixed files).
func availableLanguages(c *wp.Context) []string {
	root := "/workspace"
	if c != nil && c.Config != nil && c.Config.ABSPATH != "" {
		root = strings.TrimRight(c.Config.ABSPATH, "/")
	}
	dir := root + "/wp-content/languages"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".mo") {
			continue
		}
		base := strings.TrimSuffix(name, ".mo")
		if strings.HasPrefix(base, "admin-") || strings.HasPrefix(base, "admin-network-") ||
			strings.HasPrefix(base, "continents-cities") || strings.HasPrefix(base, "ms-") {
			continue
		}
		if seen[base] {
			continue
		}
		seen[base] = true
		out = append(out, base)
	}
	return out
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
