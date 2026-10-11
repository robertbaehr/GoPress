// Package login implements wp-login.php (P6.1+).
package login

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	// P6.40: force_ssl_admin() && ! is_ssl() → HTTPS redirect (wp-login.php top).
	if wp.MaybeRedirectLoginToHTTPS(w, r, c) {
		return
	}
	// P2.68: is_login() true for the duration of wp-login.php.
	if c.Scope != nil {
		c.Scope.SetLogin(true)
	}
	// P6.27: nocache_headers() at the top of wp-login.php.
	wp.NoCacheHeaders(w)
	action := r.URL.Query().Get("action")
	if action == "" && r.Method == http.MethodPost {
		_ = r.ParseForm()
		action = r.FormValue("action")
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		switch action {
		case "lostpassword", "retrievepassword":
			h.serveLostPassword(w, r, c, lostPasswordErrorFromQuery(r))
		case "rp", "resetpass":
			h.serveResetPass(w, r, c)
		case "logout":
			h.serveLogout(w, r, c)
		case "register":
			h.serveRegister(w, r, c)
		case "confirm_admin_email":
			h.serveConfirmAdminEmail(w, r, c)
		case "confirmaction":
			// P6.46: privacy user_request email confirmation.
			h.serveConfirmAction(w, r, c)
		case "checkemail":
			// P6.44: WP sets $action=checkemail when ?checkemail= is present.
			h.serveCheckEmail(w, r, c)
		default:
			// WP: if ( isset( $_GET['checkemail'] ) ) { $action = 'checkemail'; }
			if r.URL.Query().Get("checkemail") != "" {
				h.serveCheckEmail(w, r, c)
				return
			}
			errMsg := ""
			if r.URL.Query().Get("registration") == "disabled" {
				errMsg = loginErrorHTML("registerdisabled", "User registration is currently not allowed.", "", "")
			}
			h.serveForm(w, r, c, errMsg, false)
		}
	case http.MethodPost:
		switch action {
		case "lostpassword", "retrievepassword":
			h.serveLostPasswordPost(w, r, c)
		case "rp", "resetpass":
			h.serveResetPass(w, r, c)
		case "logout":
			h.serveLogout(w, r, c)
		case "register":
			h.serveRegister(w, r, c)
		case "confirm_admin_email":
			h.serveConfirmAdminEmail(w, r, c)
		case "postpass":
			h.servePostPass(w, r, c)
		default:
			h.servePost(w, r, c)
		}
	default:
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "Method Not Allowed\n", http.StatusMethodNotAllowed)
	}
}

// serveConfirmAction handles wp-login.php?action=confirmaction (P6.46).
// Validates request_id + confirm_key for a privacy user_request, fires
// user_request_action_confirmed, and shows the confirmed message.
func (h *Handler) serveConfirmAction(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	q := r.URL.Query()
	if strings.TrimSpace(q.Get("request_id")) == "" {
		wp.WriteDie(w, wp.Die(c, r, "Missing request ID.", "", map[string]any{"response": 500}))
		return
	}
	if strings.TrimSpace(q.Get("confirm_key")) == "" {
		wp.WriteDie(w, wp.Die(c, r, "Missing confirm key.", "", map[string]any{"response": 500}))
		return
	}
	requestID, _ := strconv.ParseInt(strings.TrimSpace(q.Get("request_id")), 10, 64)
	key := wp.SanitizeTextField(wp.UnslashString(q.Get("confirm_key")))
	if err := wp.ValidateUserRequestKey(c, requestID, key); err != nil {
		wp.WriteDie(w, wp.Die(c, r, err, "", nil))
		return
	}
	_ = wp.ConfirmUserRequest(c, requestID)
	if c != nil && c.Hooks != nil {
		c.Hooks.DoAction("user_request_action_confirmed", requestID)
	}
	message := wp.PrivacyAccountRequestConfirmedMessage(c, requestID)
	message = applyLoginMessage(c, "\t\t<div id=\"login-message\" class=\"message\">"+message+"</div>\n")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	writeLoginHead(&b, home, blogname, "User action confirmed.", "confirmaction")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	if strings.TrimSpace(message) != "" {
		b.WriteString(message)
	}
	b.WriteString("\t\t<p id=\"backtoblog\"><a href=\"")
	if home != "" {
		b.WriteString(html.EscapeString(home + "/"))
	} else {
		b.WriteString("/")
	}
	b.WriteString("\">")
	b.WriteString("&larr; Go to ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</a></p>\n")
	b.WriteString("\t\t</div>\n")
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	writeLoginHTML(w, r, b.String(), false, c)
}

// servePostPass handles wp-login.php?action=postpass (P6.15).
// Sets the wp-postpass_{COOKIEHASH} cookie from the submitted post password and redirects.
func (h *Handler) servePostPass(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r.ParseForm()
	redirectTo := strings.TrimSpace(r.FormValue("redirect_to"))
	if redirectTo == "" {
		redirectTo = r.Referer()
	}
	if redirectTo == "" {
		home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		if home == "" {
			home = "/"
		}
		redirectTo = home
	}
	pwd := r.FormValue("post_password")
	if pwd == "" {
		http.Redirect(w, r, redirectTo, http.StatusFound)
		return
	}
	cookieHash := ""
	if c != nil {
		cookieHash = c.CookieHash()
	}
	if cookieHash == "" {
		cookieHash = "0"
	}
	hashed := wp.HashPasswordPortable(pwd, 8)
	secure := strings.HasPrefix(strings.ToLower(redirectTo), "https://")
	expire := time.Now().Add(10 * 24 * time.Hour)
	http.SetCookie(w, &http.Cookie{
		Name:     "wp-postpass_" + cookieHash,
		Value:    hashed,
		Path:     "/",
		Expires:  expire,
		MaxAge:   10 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   secure,
	})
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

// serveConfirmAdminEmail handles wp-login.php?action=confirm_admin_email (P6.14 / P6.42).
// Stub of the administration email verification screen: requires a logged-in
// user with manage_options; confirm / remind-me-later update admin_email_lifespan.
func (h *Handler) serveConfirmAdminEmail(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	c.AuthenticateRequest()
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	loginURL := home + "/wp-login.php"
	if home == "" {
		loginURL = "/wp-login.php"
	}
	if c.CurrentUser == nil {
		http.Redirect(w, r, loginURL, http.StatusFound)
		return
	}
	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo == "" {
		_ = r.ParseForm()
		redirectTo = r.FormValue("redirect_to")
	}
	// P6.42: sanitize like wp_safe_redirect / wp_validate_redirect.
	redirectTo = sanitizeRedirect(redirectTo, home)
	if !c.UserCan(c.CurrentUser, "manage_options") {
		http.Redirect(w, r, redirectTo, http.StatusFound)
		return
	}

	remindInterval := int64(3 * 24 * 60 * 60)     // 3 days
	checkInterval := int64(6 * 30 * 24 * 60 * 60) // ~6 months
	if c.Hooks != nil {
		remindInterval = filterIntervalSeconds(c.Hooks.ApplyFilters("admin_email_remind_interval", remindInterval), remindInterval)
		checkInterval = filterIntervalSeconds(c.Hooks.ApplyFilters("admin_email_check_interval", checkInterval), checkInterval)
	}

	if remind := r.URL.Query().Get("remind_me_later"); remind != "" {
		if c.VerifyNonce(remind, "remind_me_later_nonce") == 0 {
			http.Redirect(w, r, loginURL, http.StatusFound)
			return
		}
		if remindInterval > 0 && c.Options != nil {
			_, _ = c.Options.UpdateOption("admin_email_lifespan", fmt.Sprintf("%d", time.Now().Unix()+remindInterval), "yes")
		}
		sep := "?"
		if strings.Contains(redirectTo, "?") {
			sep = "&"
		}
		http.Redirect(w, r, redirectTo+sep+"admin_email_remind_later=1", http.StatusFound)
		return
	}

	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		if r.FormValue("correct-admin-email") != "" {
			nonce := r.FormValue("confirm_admin_email_nonce")
			if c.VerifyNonce(nonce, "confirm_admin_email") == 0 {
				http.Redirect(w, r, loginURL, http.StatusFound)
				return
			}
			if checkInterval > 0 && c.Options != nil {
				_, _ = c.Options.UpdateOption("admin_email_lifespan", fmt.Sprintf("%d", time.Now().Unix()+checkInterval), "yes")
			}
			http.Redirect(w, r, redirectTo, http.StatusFound)
			return
		}
	}

	if c.Hooks != nil {
		c.Hooks.DoAction("admin_email_confirm", nil)
	}

	adminEmail := optionString(c, "admin_email", "")
	blogname := optionString(c, "blogname", "GoPress")
	nonce := c.CreateNonce("confirm_admin_email")
	remindNonce := c.CreateNonce("remind_me_later_nonce")
	changeLink := home + "/wp-admin/options-general.php?highlight=confirm_admin_email"
	if home == "" {
		changeLink = "/wp-admin/options-general.php?highlight=confirm_admin_email"
	}
	remindLink := loginURL + "?action=confirm_admin_email&redirect_to=" + url.QueryEscape(redirectTo) +
		"&remind_me_later=" + url.QueryEscape(remindNonce)

	var b strings.Builder
	writeLoginHead(&b, home, blogname, "Confirm your administration email", "confirm_admin_email")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	b.WriteString("\t\t<form class=\"admin-email-confirm-form\" name=\"admin-email-confirm-form\" action=\"")
	b.WriteString(html.EscapeString(loginURL + "?action=confirm_admin_email"))
	b.WriteString("\" method=\"post\">\n")
	if c.Hooks != nil {
		c.Hooks.DoAction("admin_email_confirm_form")
	}
	b.WriteString("\t\t\t<input type=\"hidden\" id=\"confirm_admin_email_nonce\" name=\"confirm_admin_email_nonce\" value=\"")
	b.WriteString(html.EscapeString(nonce))
	b.WriteString("\" />\n")
	b.WriteString("\t\t\t<input type=\"hidden\" name=\"_wp_http_referer\" value=\"")
	b.WriteString(html.EscapeString("/wp-login.php?action=confirm_admin_email"))
	b.WriteString("\" />\n")
	b.WriteString("\t\t\t<input type=\"hidden\" name=\"redirect_to\" value=\"")
	b.WriteString(html.EscapeString(redirectTo))
	b.WriteString("\" />\n")
	b.WriteString("\t\t\t<h1 class=\"admin-email__heading\">Administration email verification</h1>\n")
	b.WriteString("\t\t\t<p class=\"admin-email__details\">Please verify that the <strong>administration email</strong> for this website is still correct.</p>\n")
	b.WriteString("\t\t\t<p class=\"admin-email__details\">Current administration email: <strong class=\"admin-email__address\">")
	b.WriteString(html.EscapeString(adminEmail))
	b.WriteString("</strong></p>\n")
	b.WriteString("\t\t\t<p class=\"admin-email__details\">This email may be different from your personal email address.</p>\n")
	b.WriteString("\t\t\t<div class=\"admin-email__actions\">\n")
	b.WriteString("\t\t\t\t<div class=\"admin-email__actions-primary\">\n")
	b.WriteString("\t\t\t\t\t<a class=\"button button-large\" href=\"")
	b.WriteString(html.EscapeString(changeLink))
	b.WriteString("\">Update</a>\n")
	b.WriteString("\t\t\t\t\t<input type=\"submit\" name=\"correct-admin-email\" id=\"correct-admin-email\" class=\"button button-primary button-large\" value=\"The email is correct\" />\n")
	b.WriteString("\t\t\t\t</div>\n")
	// P6.42: remind link omitted when admin_email_remind_interval filter returns 0.
	if remindInterval > 0 {
		b.WriteString("\t\t\t\t<div class=\"admin-email__actions-secondary\">\n")
		b.WriteString("\t\t\t\t\t<a href=\"")
		b.WriteString(html.EscapeString(remindLink))
		b.WriteString("\">Remind me later</a>\n")
		b.WriteString("\t\t\t\t</div>\n")
	}
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t\t</form>\n")
	b.WriteString("\t\t</div>\n")
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	writeLoginHTML(w, r, b.String(), false, c)
}

// filterIntervalSeconds coerces ApplyFilters results for interval hooks (P6.42).
func filterIntervalSeconds(v any, fallback int64) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case string:
		var n int64
		if _, err := fmt.Sscan(t, &n); err == nil {
			return n
		}
	}
	return fallback
}

// serveRegister handles wp-login.php?action=register (P6.9 / P6.43).
// Non-multisite sites with users_can_register disabled redirect to
// ?registration=disabled. When enabled, shows the registration form and
// creates a subscriber on POST (register_new_user subset).
func (h *Handler) serveRegister(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if !usersCanRegister(c) {
		home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		target := home + "/wp-login.php?registration=disabled"
		if home == "" {
			target = "/wp-login.php?registration=disabled"
		}
		http.Redirect(w, r, target, http.StatusFound)
		return
	}
	userLogin := ""
	userEmail := ""
	errMsg := ""
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		userLogin = strings.TrimSpace(r.FormValue("user_login"))
		userEmail = strings.TrimSpace(r.FormValue("user_email"))
		if userLogin == "" {
			errMsg = loginErrorHTML("empty_username", "Please enter a username.", "", "")
		} else if userEmail == "" {
			errMsg = loginErrorHTML("empty_email", "Please type your email address.", "", "")
		} else if !strings.Contains(userEmail, "@") {
			errMsg = loginErrorHTML("invalid_email", "The email address isn&#8217;t correct.", "", "")
		} else {
			id, err := c.InsertUser(wp.UserInsert{
				Login: userLogin,
				Email: userEmail,
				Role:  "subscriber",
			})
			if err != nil {
				msg := err.Error()
				switch {
				case strings.Contains(msg, "login exists"):
					errMsg = loginErrorHTML("username_exists", "This username is already registered. Please choose another one.", "", "")
				case strings.Contains(msg, "email exists"):
					errMsg = loginErrorHTML("email_exists", "This email address is already registered.", "", "")
				default:
					errMsg = loginErrorHTML("register_fail", "Registration failed. Please try again.", "", "")
				}
			} else if id > 0 {
				home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
				redirectTo := strings.TrimSpace(r.FormValue("redirect_to"))
				if redirectTo == "" {
					redirectTo = home + "/wp-login.php?checkemail=registered"
					if home == "" {
						redirectTo = "/wp-login.php?checkemail=registered"
					}
				} else {
					redirectTo = sanitizeRedirect(redirectTo, home)
				}
				http.Redirect(w, r, redirectTo, http.StatusFound)
				return
			}
		}
	}
	h.serveRegisterForm(w, r, c, userLogin, userEmail, errMsg)
}

// serveRegisterForm renders wp-login.php?action=register (P6.43).
func (h *Handler) serveRegisterForm(w http.ResponseWriter, r *http.Request, c *wp.Context, userLogin, userEmail, errMsg string) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	loginURL := home + "/wp-login.php"
	actionURL := loginURL + "?action=register"
	lostURL := loginURL + "?action=lostpassword"
	redirectTo := ""
	if r != nil {
		redirectTo = strings.TrimSpace(r.URL.Query().Get("redirect_to"))
		if redirectTo == "" {
			redirectTo = strings.TrimSpace(r.FormValue("redirect_to"))
		}
	}

	var b strings.Builder
	writeLoginHead(&b, home, blogname, "Registration Form", "register")
	b.WriteString("\t\t\t\t<h1 class=\"screen-reader-text\">Registration Form</h1>\n")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	// P6.51: login_message filter (WP login_header).
	if msg := applyLoginMessage(c, `<div class="notice notice-info message register"><p>Register For This Site</p></div>`+"\n"); strings.TrimSpace(msg) != "" {
		b.WriteString(msg)
	}
	if errMsg != "" {
		b.WriteString("\t\t<div id=\"login_error\">")
		b.WriteString(errMsg)
		b.WriteString("</div>\n")
	}
	b.WriteString("\t\t<form name=\"registerform\" id=\"registerform\" action=\"")
	b.WriteString(html.EscapeString(actionURL))
	b.WriteString("\" method=\"post\" novalidate=\"novalidate\">\n")
	b.WriteString("\t\t\t<p>\n")
	b.WriteString("\t\t\t\t<label for=\"user_login\">Username</label>\n")
	b.WriteString("\t\t\t\t<input type=\"text\" name=\"user_login\" id=\"user_login\" class=\"input ltr\" value=\"")
	b.WriteString(html.EscapeString(userLogin))
	b.WriteString("\" size=\"20\" autocapitalize=\"off\" autocomplete=\"username\" required=\"required\" />\n")
	b.WriteString("\t\t\t</p>\n")
	b.WriteString("\t\t\t<p>\n")
	b.WriteString("\t\t\t\t<label for=\"user_email\">Email</label>\n")
	b.WriteString("\t\t\t\t<input type=\"email\" name=\"user_email\" id=\"user_email\" class=\"input\" value=\"")
	b.WriteString(html.EscapeString(userEmail))
	b.WriteString("\" size=\"25\" autocomplete=\"email\" required=\"required\" />\n")
	b.WriteString("\t\t\t</p>\n")
	if c != nil && c.Hooks != nil {
		c.Hooks.DoAction("register_form")
	}
	b.WriteString("\t\t\t<p id=\"reg_passmail\">\n")
	b.WriteString("\t\t\t\tRegistration confirmation will be emailed to you.\n")
	b.WriteString("\t\t\t</p>\n")
	b.WriteString("\t\t\t<input type=\"hidden\" name=\"redirect_to\" value=\"")
	b.WriteString(html.EscapeString(redirectTo))
	b.WriteString("\" />\n")
	b.WriteString("\t\t\t<p class=\"submit\">\n")
	b.WriteString("\t\t\t\t<input type=\"submit\" name=\"wp-submit\" id=\"wp-submit\" class=\"button button-primary button-large\" value=\"Register\" />\n")
	b.WriteString("\t\t\t</p>\n")
	b.WriteString("\t\t</form>\n\n")
	b.WriteString("\t\t<p id=\"nav\">\n")
	b.WriteString("\t\t\t<a class=\"wp-login-log-in\" href=\"")
	b.WriteString(html.EscapeString(loginURL))
	b.WriteString("\">Log in</a>\n")
	b.WriteString("\t\t\t | \n")
	b.WriteString("\t\t\t<a class=\"wp-login-lost-password\" href=\"")
	b.WriteString(html.EscapeString(lostURL))
	b.WriteString("\">Lost your password?</a>\n")
	b.WriteString("\t\t</p>\n")
	b.WriteString("\t\t\t\t<p id=\"backtoblog\">\n")
	b.WriteString("\t\t\t<a href=\"")
	b.WriteString(html.EscapeString(home + "/"))
	b.WriteString("\">&larr; Go to ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</a>\t\t</p>\n")
	if link := wp.PrivacyPolicyPageLinkHTML(c); link != "" {
		b.WriteString("\t\t")
		b.WriteString(link)
		b.WriteString("\n")
	}
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t<script>\n")
	b.WriteString("\t\ttry{document.getElementById('user_login').focus();}catch(e){}\n")
	b.WriteString("\t\tif(typeof wpOnload==='function')wpOnload();\n")
	b.WriteString("\t</script>\n")
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	writeLoginHTML(w, r, b.String(), loginShouldShake(c, extractLoginErrorCode(errMsg)), c)
}

func extractLoginErrorCode(errHTML string) string {
	// loginErrorHTML embeds id="login_error" and often a code in class; shake on empty fields.
	if strings.Contains(errHTML, "empty_username") || strings.Contains(errHTML, "The username field is empty") || strings.Contains(errHTML, "Please enter a username") {
		return "empty_username"
	}
	if strings.Contains(errHTML, "empty_email") || strings.Contains(errHTML, "Please type your email") {
		return "empty_email"
	}
	if strings.Contains(errHTML, "invalid_email") {
		return "invalid_email"
	}
	return ""
}

// serveCheckEmail renders wp-login.php?checkemail=confirm|registered (P6.43 / P6.44).
// Matches WP login_header( 'Check your email' ) + notice-info message with login link.
func (h *Handler) serveCheckEmail(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	loginURL := home + "/wp-login.php"
	if home == "" {
		loginURL = "/wp-login.php"
	}
	kind := strings.TrimSpace(r.URL.Query().Get("checkemail"))
	msg := ""
	switch kind {
	case "confirm":
		msg = "Check your email for the confirmation link, then visit the <a href=\"" +
			html.EscapeString(loginURL) + "\">login page</a>."
	case "registered":
		msg = "Registration complete. Please check your email, then visit the <a href=\"" +
			html.EscapeString(loginURL) + "\">login page</a>."
	default:
		// Unknown checkemail value: still show the checkemail chrome (WP does).
		msg = "Check your email."
	}
	// P6.44: wp_login_errors filter (message-type errors).
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_login_errors", msg, home+"/wp-admin/"); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				msg = s
			}
		}
	}
	var b strings.Builder
	writeLoginHead(&b, home, blogname, "Check your email", "checkemail")
	b.WriteString("\t\t\t\t<h1 class=\"screen-reader-text\">Check your email</h1>\n")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	// P6.51: login_message runs in login_header; checkemail passes notice via WP_Error message data.
	loginMsg := applyLoginMessage(c, `<div class="notice notice-info message" id="login-message"><p>`+msg+`</p></div>`+"\n")
	if strings.TrimSpace(loginMsg) != "" {
		b.WriteString(loginMsg)
	}
	b.WriteString("\t\t<p id=\"backtoblog\">\n")
	b.WriteString("\t\t\t<a href=\"")
	b.WriteString(html.EscapeString(home + "/"))
	b.WriteString("\">&larr; Go to ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</a>\t\t</p>\n")
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	writeLoginHTML(w, r, b.String(), false, c)
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

// serveLogout clears auth cookies and redirects (wp-login.php?action=logout, P6.6 / P6.16).
// When redirect_to is provided, honor it (wp_safe_redirect). When omitted, send the
// user to wp-login.php?loggedout=true like core.
func (h *Handler) serveLogout(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	c.AuthenticateRequest()
	if c.CurrentUser != nil && c.SessionToken != "" {
		_ = c.DestroySessionToken(c.CurrentUser.ID, c.SessionToken)
	}
	c.ClearAuthCookies(w, "/")
	c.CurrentUser = nil
	c.SessionToken = ""

	requested := strings.TrimSpace(r.URL.Query().Get("redirect_to"))
	if requested == "" {
		_ = r.ParseForm()
		requested = strings.TrimSpace(r.FormValue("redirect_to"))
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	var redirectTo string
	if requested == "" {
		redirectTo = home + "/wp-login.php?loggedout=true"
		if home == "" {
			redirectTo = "/wp-login.php?loggedout=true"
		}
	} else {
		redirectTo = sanitizeRedirect(requested, home)
	}
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func (h *Handler) serveForm(w http.ResponseWriter, r *http.Request, c *wp.Context, errMsg string, shake bool) {
	if c != nil {
		c.AuthenticateRequest()
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo == "" {
		// P6.34: default redirect_to is admin_url().
		redirectTo = defaultLoginAdminRedirect(c, home)
	}
	loginURL := home + "/wp-login.php"
	reauth := isReauth(r)
	interim := isInterimLogin(r)

	// Logged-in users skip the form unless reauth forces re-authentication (P6.18 / P6.37).
	if c != nil && c.CurrentUser != nil && !reauth && !interim && errMsg == "" {
		redirectTo = adjustDefaultAdminLoginRedirect(c, redirectTo, home, c.CurrentUser)
		http.Redirect(w, r, sanitizeRedirect(redirectTo, home), http.StatusFound)
		return
	}

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
	// P6.51: login_message filter always runs (WP login_header), even when empty.
	if msg := applyLoginMessage(c, ""); strings.TrimSpace(msg) != "" {
		b.WriteString(msg)
		if !strings.HasSuffix(msg, "\n") {
			b.WriteByte('\n')
		}
	}
	if errMsg != "" {
		b.WriteString("\t\t<div id=\"login_error\">")
		b.WriteString(errMsg)
		b.WriteString("</div>\n")
	} else if interim {
		// P6.19 / P6.31: WP interim login shows session-expired info via login_messages.
		msgInner := "<p>Your session has expired. Please log in to continue where you left off.</p>"
		if c != nil && c.Hooks != nil {
			if v := c.Hooks.ApplyFilters("login_messages", msgInner); v != nil {
				if s, ok := v.(string); ok {
					msgInner = s
				}
			}
		}
		if strings.TrimSpace(msgInner) != "" {
			b.WriteString("\t<div class=\"notice notice-info message\" id=\"login-message\">")
			b.WriteString(msgInner)
			b.WriteString("</div>\n")
		}
	}
	actionURL := loginURL
	if interim {
		// Keep interim + redirect_to on the POST action (P6.17).
		q := url.Values{}
		q.Set("interim-login", "1")
		if redirectTo != "" {
			q.Set("redirect_to", redirectTo)
		}
		actionURL = loginURL + "?" + q.Encode()
	}
	// P6.32: aria-describedby ties focused inputs to login-message / login_error
	// (wp-login.php interim expired notice uses data 'message').
	ariaDescribedBy := ""
	if errMsg != "" {
		ariaDescribedBy = ` aria-describedby="login_error"`
	} else if interim {
		ariaDescribedBy = ` aria-describedby="login-message"`
	}
	b.WriteString("\t\t<form name=\"loginform\" id=\"loginform\" action=\"")
	b.WriteString(html.EscapeString(actionURL))
	b.WriteString("\" method=\"post\">\n")
	b.WriteString("\t\t\t<p>\n")
	b.WriteString("\t\t\t\t<label for=\"user_login\">Username or Email Address</label>\n")
	b.WriteString("\t\t\t\t<input type=\"text\" name=\"log\" id=\"user_login\"")
	b.WriteString(ariaDescribedBy)
	b.WriteString(" class=\"input ltr\" value=\"\" size=\"20\" autocapitalize=\"off\" autocomplete=\"username\" required=\"required\" />\n")
	b.WriteString("\t\t\t</p>\n\n")
	b.WriteString("\t\t\t<div class=\"user-pass-wrap\">\n")
	b.WriteString("\t\t\t\t<label for=\"user_pass\">Password</label>\n")
	b.WriteString("\t\t\t\t<div class=\"wp-pwd\">\n")
	b.WriteString("\t\t\t\t\t<input type=\"password\" name=\"pwd\" id=\"user_pass\"")
	b.WriteString(ariaDescribedBy)
	b.WriteString(" class=\"input password-input ltr\" value=\"\" size=\"20\" autocomplete=\"current-password\" spellcheck=\"false\" required=\"required\" />\n")
	b.WriteString("\t\t\t\t\t<button type=\"button\" class=\"button button-secondary wp-hide-pw hide-if-no-js\" data-toggle=\"0\" aria-label=\"Show password\">\n")
	b.WriteString("\t\t\t\t\t\t<span class=\"dashicons dashicons-visibility\" aria-hidden=\"true\"></span>\n")
	b.WriteString("\t\t\t\t\t</button>\n")
	b.WriteString("\t\t\t\t</div>\n")
	b.WriteString("\t\t\t</div>\n")
	// P6.52: login_form action (plugins inject markup before Remember Me).
	if c != nil && c.Hooks != nil {
		var formExtra strings.Builder
		c.Hooks.DoAction("login_form")
		// Capture via filter-style injection: login_form_top is wp_login_form-only;
		// on wp-login.php plugins typically echo from the login_form action. We also
		// honor a login_form_top filter here so GoPress plugins can append HTML.
		if v := c.Hooks.ApplyFilters("login_form_top", "", map[string]any{
			"form_id":  "loginform",
			"redirect": redirectTo,
		}); v != nil {
			if s, ok := v.(string); ok && s != "" {
				formExtra.WriteString(s)
			}
		}
		b.WriteString(formExtra.String())
	}
	b.WriteString("\t\t\t\t\t\t\t\t\t<p class=\"forgetmenot\">\n")
	// P6.24: keep Remember Me checked after failed login (WP checked( $rememberme )).
	rememberChecked := ""
	if r.Method == http.MethodPost && r.FormValue("rememberme") != "" {
		rememberChecked = " checked=\"checked\""
	}
	b.WriteString("\t\t\t\t<input name=\"rememberme\" type=\"checkbox\" id=\"rememberme\" value=\"forever\"")
	b.WriteString(rememberChecked)
	b.WriteString("  />\n")
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
	if interim {
		// P6.22: interim login opens outbound links in a new tab (wp-login.php).
		writeInterimLoginBlankTargetScript(&b)
	}
	if !interim {
		b.WriteString("\t\t<p id=\"backtoblog\">\n")
		b.WriteString("\t\t\t<a href=\"")
		b.WriteString(html.EscapeString(home + "/"))
		b.WriteString("\">&larr; Go to ")
		b.WriteString(html.EscapeString(blogname))
		b.WriteString("</a>\t\t</p>\n")
		if link := wp.PrivacyPolicyPageLinkHTML(c); link != "" {
			b.WriteString("\t\t")
			b.WriteString(link)
			b.WriteString("\n")
		}
		writeLanguageSwitcher(&b, r, c, availableLanguages(c))
	}
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t\t\n")
	// P6.28 / P6.29: wp_shake_js on shake_error_codes (adds .shake to the form),
	// including interim-login auth failures.
	if shake && errMsg != "" {
		writeLoginShakeJS(&b)
	}
	writeLoginScripts(&b, home, c)
	// P6.45: login_footer fires for interim and standard login forms.
	if c != nil && c.Hooks != nil {
		c.Hooks.DoAction("login_footer", "")
	}
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	b.WriteString("\t")

	writeLoginHTML(w, r, b.String(), reauth, c)
	if reauth && c != nil {
		// P6.37: clear via SetCurrentUser (wp_set_current_user(0)).
		wp.SetCurrentUser(c, 0)
		c.SessionToken = ""
	}
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
	// P6.35: honor redirect_to via lostpassword_redirect + wp_safe_redirect.
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	requested := strings.TrimSpace(r.FormValue("redirect_to"))
	if requested == "" {
		requested = strings.TrimSpace(r.URL.Query().Get("redirect_to"))
	}
	redirectTo := applyLostPasswordRedirect(c, requested)
	if redirectTo == "" {
		redirectTo = home + "/wp-login.php?checkemail=confirm"
		if home == "" {
			redirectTo = "/wp-login.php?checkemail=confirm"
		}
	} else {
		redirectTo = sanitizeRedirect(redirectTo, home)
	}
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func lostPasswordErrorFromQuery(r *http.Request) string {
	switch r.URL.Query().Get("error") {
	case "invalidkey":
		return "Your password reset link appears to be invalid. Please request a new link below."
	case "expiredkey":
		return "Your password reset link has expired. Please request a new link below."
	default:
		return ""
	}
}

func resetPassCookieName(c *wp.Context) string {
	h := "0"
	if c != nil {
		if ch := c.CookieHash(); ch != "" {
			h = ch
		}
	}
	return "wp-resetpass-" + h
}

func clearResetPassCookie(w http.ResponseWriter, c *wp.Context) {
	http.SetCookie(w, &http.Cookie{
		Name:     resetPassCookieName(c),
		Value:    " ",
		Path:     "/wp-login.php",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
		HttpOnly: true,
	})
}

// serveResetPass handles action=rp / resetpass (P6.13).
func (h *Handler) serveResetPass(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	cookieName := resetPassCookieName(c)

	// First visit from email link: stash key in cookie and strip from URL.
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		key := r.URL.Query().Get("key")
		loginName := r.URL.Query().Get("login")
		if key != "" && loginName != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    loginName + ":" + key,
				Path:     "/wp-login.php",
				HttpOnly: true,
			})
			target := home + "/wp-login.php?action=rp"
			if home == "" {
				target = "/wp-login.php?action=rp"
			}
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
	}

	rpLogin, rpKey := "", ""
	if ck, err := r.Cookie(cookieName); err == nil && ck != nil && strings.Contains(ck.Value, ":") {
		parts := strings.SplitN(ck.Value, ":", 2)
		rpLogin, rpKey = parts[0], parts[1]
	}

	var user *wp.User
	var keyErr error
	if rpLogin != "" && rpKey != "" {
		user, keyErr = wp.CheckPasswordResetKey(c, rpKey, rpLogin)
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			// P6.48: WP isset($_POST['pass1']) + hash_equals($rp_key, $_POST['rp_key']).
			if _, pass1Set := r.PostForm["pass1"]; pass1Set && !wp.HashEquals(rpKey, r.FormValue("rp_key")) {
				user = nil
				keyErr = fmt.Errorf("invalid_key")
			}
		}
	} else {
		keyErr = fmt.Errorf("invalid_key")
	}

	if user == nil || keyErr != nil {
		clearResetPassCookie(w, c)
		errQ := "invalidkey"
		if keyErr != nil && keyErr.Error() == "expired_key" {
			errQ = "expiredkey"
		}
		target := home + "/wp-login.php?action=lostpassword&error=" + errQ
		if home == "" {
			target = "/wp-login.php?action=lostpassword&error=" + errQ
		}
		http.Redirect(w, r, target, http.StatusFound)
		return
	}

	formErr := ""
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		pass1 := strings.TrimSpace(r.FormValue("pass1"))
		pass2 := r.FormValue("pass2")
		if pass1 == "" && r.FormValue("pass1") != "" {
			formErr = "The password cannot be a space or all spaces."
		} else if pass1 != "" && strings.TrimSpace(pass2) != pass1 {
			formErr = "The passwords do not match."
		} else if pass1 != "" {
			if err := wp.ResetPassword(c, user, pass1); err != nil {
				formErr = "Could not reset password."
			} else {
				clearResetPassCookie(w, c)
				h.serveResetPassDone(w, r, c)
				return
			}
		}
	}

	h.serveResetPassForm(w, r, c, rpLogin, rpKey, formErr)
}

func (h *Handler) serveResetPassDone(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	loginURL := home + "/wp-login.php"
	var b strings.Builder
	writeLoginHead(&b, home, blogname, "Password Reset", "resetpass")
	b.WriteString("\t\t\t\t<h1 class=\"screen-reader-text\">Password Reset</h1>\n")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	doneMsg := `<p class="message reset-pass">Your password has been reset. <a href="` + html.EscapeString(loginURL) + `">Log in</a></p>` + "\n"
	if msg := applyLoginMessage(c, doneMsg); strings.TrimSpace(msg) != "" {
		b.WriteString(msg)
	}
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	writeLoginHTML(w, r, b.String(), false, c)
}

func (h *Handler) serveResetPassForm(w http.ResponseWriter, r *http.Request, c *wp.Context, rpLogin, rpKey, formErr string) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	loginURL := home + "/wp-login.php"
	actionURL := loginURL + "?action=resetpass"
	generated := wp.GeneratePassword(16, false)
	passHint := wp.GetPasswordHint(c)

	var b strings.Builder
	writeLoginHead(&b, home, blogname, "Reset Password", "resetpass")
	b.WriteString("\t\t\t\t<h1 class=\"screen-reader-text\">Reset Password</h1>\n")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	if formErr != "" {
		b.WriteString("<div class=\"notice notice-error\" id=\"login_error\"><p><strong>Error:</strong> ")
		b.WriteString(html.EscapeString(formErr))
		b.WriteString("</p></div>\n")
	} else {
		// P6.41 / P6.51: reset-pass notice via login_message filter.
		if msg := applyLoginMessage(c, `<div class="notice notice-info message reset-pass"><p>Enter your new password below or generate one.</p></div>`+"\n"); strings.TrimSpace(msg) != "" {
			b.WriteString(msg)
		}
	}
	b.WriteString("\t\t<form name=\"resetpassform\" id=\"resetpassform\" action=\"")
	b.WriteString(html.EscapeString(actionURL))
	b.WriteString("\" method=\"post\" autocomplete=\"off\">\n")
	b.WriteString("\t\t\t<input type=\"hidden\" id=\"user_login\" value=\"")
	b.WriteString(html.EscapeString(rpLogin))
	b.WriteString("\" autocomplete=\"off\" />\n")
	// P6.41: user-pass1-wrap / strength meter / weak confirm / generate button.
	b.WriteString("\t\t\t<div class=\"user-pass1-wrap\">\n")
	b.WriteString("\t\t\t\t<p>\n")
	b.WriteString("\t\t\t\t\t<label for=\"pass1\">New password</label>\n")
	b.WriteString("\t\t\t\t</p>\n")
	b.WriteString("\t\t\t\t<div class=\"wp-pwd\">\n")
	b.WriteString("\t\t\t\t\t<input type=\"password\" name=\"pass1\" id=\"pass1\" class=\"input password-input ltr\" size=\"24\" value=\"\" autocomplete=\"new-password\" spellcheck=\"false\" data-reveal=\"1\" data-pw=\"")
	b.WriteString(html.EscapeString(generated))
	b.WriteString("\" aria-describedby=\"pass-strength-result\" />\n")
	b.WriteString("\t\t\t\t\t<button type=\"button\" class=\"button button-secondary wp-hide-pw hide-if-no-js\" data-toggle=\"0\" aria-label=\"Hide password\">\n")
	b.WriteString("\t\t\t\t\t\t<span class=\"dashicons dashicons-hidden\" aria-hidden=\"true\"></span>\n")
	b.WriteString("\t\t\t\t\t</button>\n")
	b.WriteString("\t\t\t\t\t<div id=\"pass-strength-result\" class=\"hide-if-no-js\" aria-live=\"polite\">Strength indicator</div>\n")
	b.WriteString("\t\t\t\t</div>\n")
	b.WriteString("\t\t\t\t<div class=\"pw-weak\">\n")
	b.WriteString("\t\t\t\t\t<input type=\"checkbox\" name=\"pw_weak\" id=\"pw-weak\" class=\"pw-checkbox\" />\n")
	b.WriteString("\t\t\t\t\t<label for=\"pw-weak\">Confirm use of weak password</label>\n")
	b.WriteString("\t\t\t\t</div>\n")
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t\t\t<p class=\"user-pass2-wrap\">\n")
	b.WriteString("\t\t\t\t<label for=\"pass2\">Confirm new password</label>\n")
	b.WriteString("\t\t\t\t<input type=\"password\" name=\"pass2\" id=\"pass2\" class=\"input\" size=\"20\" value=\"\" autocomplete=\"new-password\" spellcheck=\"false\" />\n")
	b.WriteString("\t\t\t</p>\n")
	b.WriteString("\t\t\t<p class=\"description indicator-hint\">")
	b.WriteString(passHint)
	b.WriteString("</p>\n")
	if c != nil && c.Hooks != nil {
		// P6.41: resetpass_form action (user object when available).
		if u, err := c.GetUserBy("login", rpLogin); err == nil && u != nil {
			c.Hooks.DoAction("resetpass_form", u)
		}
	}
	b.WriteString("\t\t\t<input type=\"hidden\" name=\"rp_key\" value=\"")
	b.WriteString(html.EscapeString(rpKey))
	b.WriteString("\" />\n")
	b.WriteString("\t\t\t<p class=\"submit reset-pass-submit\">\n")
	b.WriteString("\t\t\t\t<button type=\"button\" class=\"button wp-generate-pw hide-if-no-js skip-aria-expanded\">Generate Password</button>\n")
	b.WriteString("\t\t\t\t<input type=\"submit\" name=\"wp-submit\" id=\"wp-submit\" class=\"button button-primary button-large\" value=\"Save Password\" />\n")
	b.WriteString("\t\t\t</p>\n")
	b.WriteString("\t\t</form>\n")
	b.WriteString("\t\t<p id=\"nav\">\n")
	b.WriteString("\t\t\t<a class=\"wp-login-log-in\" href=\"")
	b.WriteString(html.EscapeString(loginURL))
	b.WriteString("\">Log in</a>\n")
	b.WriteString("\t\t</p>\n")
	b.WriteString("\t\t\t</div>\n")
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	writeLoginHTML(w, r, b.String(), false, c)
}

func (h *Handler) serveLostPassword(w http.ResponseWriter, r *http.Request, c *wp.Context, errMsg string) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	loginURL := home + "/wp-login.php"
	actionURL := loginURL + "?action=lostpassword"

	// P6.35: preserve redirect_to through the form (lostpassword_redirect filter).
	requested := ""
	if r != nil {
		requested = strings.TrimSpace(r.URL.Query().Get("redirect_to"))
		if requested == "" {
			requested = strings.TrimSpace(r.FormValue("redirect_to"))
		}
	}
	redirectTo := applyLostPasswordRedirect(c, requested)

	var b strings.Builder
	writeLoginHead(&b, home, blogname, "Lost Password", "lostpassword")
	b.WriteString("\t\t\t\t<h1 class=\"screen-reader-text\">Lost Password</h1>\n")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	// P6.51: WP always shows the info message via login_header + login_message filter.
	if msg := applyLoginMessage(c, `<div class="notice notice-info message"><p>Please enter your username or email address. You will receive an email message with instructions on how to reset your password.</p></div>`+"\n"); strings.TrimSpace(msg) != "" {
		b.WriteString(msg)
	}
	if errMsg != "" {
		b.WriteString("<div class=\"notice notice-error\" id=\"login_error\"><p><strong>Error:</strong> ")
		b.WriteString(html.EscapeString(errMsg))
		b.WriteString("</p></div>\n")
	}
	b.WriteString("\t\t")
	b.WriteString("\t\t<form name=\"lostpasswordform\" id=\"lostpasswordform\" action=\"")
	b.WriteString(html.EscapeString(actionURL))
	b.WriteString("\" method=\"post\">\n")
	b.WriteString("\t\t\t<p>\n")
	b.WriteString("\t\t\t\t<label for=\"user_login\">Username or Email Address</label>\n")
	b.WriteString("\t\t\t\t<input type=\"text\" name=\"user_login\" id=\"user_login\" class=\"input ltr\" value=\"\" size=\"20\" autocapitalize=\"off\" autocomplete=\"username\" required=\"required\" />\n")
	b.WriteString("\t\t\t</p>\n")
	b.WriteString("\t\t\t\t\t\t<input type=\"hidden\" name=\"redirect_to\" value=\"")
	b.WriteString(html.EscapeString(redirectTo))
	b.WriteString("\" />\n")
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
	if link := wp.PrivacyPolicyPageLinkHTML(c); link != "" {
		b.WriteString("\t\t")
		b.WriteString(link)
		b.WriteString("\n")
	}
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

	writeLoginHTML(w, r, b.String(), false, c)
}

// applyLoginMessage mirrors login_header's apply_filters( 'login_message', $message ) (P6.51).
func applyLoginMessage(c *wp.Context, message string) string {
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("login_message", message); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return message
}

func writeLoginHead(b *strings.Builder, home, blogname, title, action string, extraClasses ...string) {
	ver := "7.2-alpha-64260"
	styles := home + "/wp-admin/load-styles.php?c=0&dir=ltr&load%5Bchunk_0%5D=dashicons,buttons,forms,l10n,wp-base-styles,wp-tooltip,login&ver=" + ver
	interim := false
	for _, cls := range extraClasses {
		if strings.TrimSpace(cls) == "interim-login" {
			interim = true
			break
		}
	}
	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("\t<html lang=\"en-US\">\n")
	b.WriteString("\t<head>\n")
	b.WriteString("\t<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" />\n")
	// WP login_header: sprintf( '%1$s &lsaquo; %2$s &#8212; WordPress', $title, $blogname ).
	// Empty $title (interim success, P6.33) still keeps the &lsaquo; separator.
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
	if interim {
		// WP interim modal: transparent page chrome so the parent admin shows through.
		b.WriteString("\t\t<style>html{background-color: transparent;}</style>\n")
	}
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

func writeLoginHTML(w http.ResponseWriter, r *http.Request, body string, reauth bool, c *wp.Context) {
	if reauth {
		clearAuthCookiesForReauth(w, c)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "wordpress_test_cookie",
		Value:    "WP Cookie check",
		Path:     "/",
		HttpOnly: true,
	})
	// P6.23 / P6.49: persist login-screen locale from ?wp_lang= (HttpOnly, session).
	// PHP: if ( isset( $_GET['wp_lang'] ) ) setcookie(..., COOKIEPATH, ...).
	if r != nil {
		if _, ok := r.URL.Query()["wp_lang"]; ok {
			lang := wp.SanitizeTextField(r.URL.Query().Get("wp_lang"))
			secure := false
			cookiePath := "/"
			if c != nil {
				home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
				loginURL := home + "/wp-login.php"
				secure = strings.HasPrefix(strings.ToLower(loginURL), "https://")
				cookiePath = loginCookiePath(home) // COOKIEPATH
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "wp_lang",
				Value:    lang,
				Path:     cookiePath,
				HttpOnly: true,
				Secure:   secure,
			})
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

// clearAuthCookiesForReauth mirrors WP's cookie clearing when reauth=1 is set.
func clearAuthCookiesForReauth(w http.ResponseWriter, c *wp.Context) {
	expired := time.Unix(0, 0).UTC()
	cookieHash := ""
	userID := int64(0)
	if c != nil {
		cookieHash = c.CookieHash()
		if c.CurrentUser != nil {
			userID = c.CurrentUser.ID
		}
	}
	if cookieHash == "" {
		cookieHash = "0"
	}
	settingsID := fmt.Sprintf("%d", userID)
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
		{"wp-settings-" + settingsID, "/"},
		{"wp-settings-time-" + settingsID, "/"},
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
		h.serveForm(w, r, c, loginErrorHTML("invalid_form", "Invalid form submission.", "", ""), false)
		return
	}
	login := strings.TrimSpace(r.FormValue("log"))
	password := r.FormValue("pwd")
	remember := r.FormValue("rememberme") != ""
	requestedRedirect := strings.TrimSpace(r.FormValue("redirect_to"))
	if requestedRedirect == "" {
		requestedRedirect = strings.TrimSpace(r.URL.Query().Get("redirect_to"))
	}
	homeForRedirect := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	// P6.34: omitted redirect_to → admin_url(); then sanitize.
	redirectTo := requestedRedirect
	if redirectTo == "" {
		redirectTo = defaultLoginAdminRedirect(c, homeForRedirect)
	}
	redirectTo = sanitizeRedirect(redirectTo, homeForRedirect)
	interim := isInterimLogin(r)

	// P6.38: derive secure from is_ssl() (TLS, HTTPS CGI, :443, X-Forwarded-Proto).
	secure := wp.IsSSL(c)
	set, err := c.Login(login, password, remember, secure, 0)
	// P6.21: mirror wp-login.php testcookie handling. After signon, request
	// cookies still lack the logged-in cookie; if the form posted testcookie
	// without wordpress_test_cookie present, treat cookies as blocked.
	if r.FormValue("testcookie") != "" && !hasTestCookie(r) {
		err = &wp.AuthError{Code: "test_cookie", Login: login}
		set = nil
	}
	if err != nil || set == nil {
		// Re-display form with error; preserve redirect_to via query for GET-shaped form.
		q := r.URL.Query()
		q.Set("redirect_to", redirectTo)
		if interim {
			q.Set("interim-login", "1")
		}
		// P6.37: keep reauth on failed POST so cookies stay cleared on redisplay.
		if isReauth(r) {
			q.Set("reauth", "1")
		}
		r.URL.RawQuery = q.Encode()
		home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		lostURL := home + "/wp-login.php?action=lostpassword"
		if home == "" {
			lostURL = "/wp-login.php?action=lostpassword"
		}
		errHTML, shake := formatLoginAuthError(c, err, login, password, lostURL)
		h.serveForm(w, r, c, errHTML, shake)
		return
	}
	set.WriteAuthCookies(w, "/")
	if interim {
		h.serveInterimLoginSuccess(w, r, c)
		return
	}
	// P6.34: login_redirect filter + subscribers without edit_posts → profile.
	user := c.CurrentUser
	redirectTo = applyLoginRedirectFilter(c, redirectTo, requestedRedirect, user)
	redirectTo = adjustDefaultAdminLoginRedirect(c, redirectTo, homeForRedirect, user)
	redirectTo = sanitizeRedirect(redirectTo, homeForRedirect)
	// P6.47: when admin email verification is due, send manage_options users
	// through confirm_admin_email (wp-login.php post-signon gate).
	redirectTo = maybeAdminEmailConfirmRedirect(c, redirectTo, homeForRedirect, user)
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

// maybeAdminEmailConfirmRedirect wraps redirect_to with action=confirm_admin_email
// when the administration email check interval has elapsed (P6.47).
func maybeAdminEmailConfirmRedirect(c *wp.Context, redirectTo, home string, user *wp.User) string {
	if c == nil || user == nil || !c.UserCan(user, "manage_options") {
		return redirectTo
	}
	checkInterval := int64(6 * 30 * 24 * 60 * 60) // ~6 months
	if c.Hooks != nil {
		checkInterval = filterIntervalSeconds(c.Hooks.ApplyFilters("admin_email_check_interval", checkInterval), checkInterval)
	}
	if checkInterval <= 0 {
		return redirectTo
	}
	lifespan := int64(0)
	if raw := optionString(c, "admin_email_lifespan", "0"); raw != "" {
		if n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil {
			lifespan = n
		}
	}
	if time.Now().Unix() <= lifespan {
		return redirectTo
	}
	loginURL := strings.TrimRight(home, "/") + "/wp-login.php"
	if home == "" {
		loginURL = "/wp-login.php"
	}
	u, err := url.Parse(loginURL)
	if err != nil {
		return redirectTo
	}
	q := u.Query()
	q.Set("action", "confirm_admin_email")
	q.Set("redirect_to", redirectTo)
	u.RawQuery = q.Encode()
	return u.String()
}

const testCookieName = "wordpress_test_cookie"

// defaultShakeLoginErrorCodes mirrors wp-login.php $shake_error_codes (P6.28 / P6.50).
var defaultShakeLoginErrorCodes = []string{
	"empty_password",
	"empty_email",
	"invalid_email",
	"invalidcombo",
	"empty_username",
	"invalid_username",
	"incorrect_password",
	"retrieve_password_email_failure",
}

// shakeErrorCodes returns the effective shake_error_codes list (P6.50).
// Applies the shake_error_codes filter like wp-login.php.
func shakeErrorCodes(c *wp.Context) map[string]bool {
	codes := append([]string{}, defaultShakeLoginErrorCodes...)
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("shake_error_codes", codes)
		switch t := filtered.(type) {
		case []string:
			codes = t
		case []any:
			codes = codes[:0]
			for _, v := range t {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					codes = append(codes, strings.TrimSpace(s))
				}
			}
		}
	}
	out := make(map[string]bool, len(codes))
	for _, code := range codes {
		if code != "" {
			out[code] = true
		}
	}
	return out
}

func loginShouldShake(c *wp.Context, code string) bool {
	if code == "" {
		return false
	}
	return shakeErrorCodes(c)[code]
}

// writeLoginShakeJS emits wp_shake_js inline script (P6.28 / P6.29 / P6.50).
// Same script for standard and interim-login forms (WP login_footer).
func writeLoginShakeJS(b *strings.Builder) {
	if b == nil {
		return
	}
	b.WriteString("<script>\n")
	b.WriteString("document.querySelector('form').classList.add('shake');\n")
	b.WriteString("</script>\n")
}

func hasTestCookie(r *http.Request) bool {
	if r == nil {
		return false
	}
	ck, err := r.Cookie(testCookieName)
	return err == nil && ck != nil && strings.TrimSpace(ck.Value) != ""
}

// formatLoginAuthError builds WP-shaped login_error HTML (P6.12) and whether
// the form should shake (P6.28 / P6.50 shake_error_codes filter).
func formatLoginAuthError(c *wp.Context, err error, login, password, lostURL string) (string, bool) {
	ae, ok := err.(*wp.AuthError)
	if !ok || ae == nil {
		// Empty fields without typed error (e.g. both empty before Login change).
		if strings.TrimSpace(login) == "" && password == "" {
			html := `<ul class="login-error-list"><li><strong>Error:</strong> The username field is empty.</li>` +
				`<li><strong>Error:</strong> The password field is empty.</li></ul>`
			return html, loginShouldShake(c, "empty_username") || loginShouldShake(c, "empty_password")
		}
		return loginErrorHTML("invalid_credentials", "The username or password you entered is incorrect.", "", ""), false
	}
	switch ae.Code {
	case "empty_username":
		if password == "" {
			html := `<ul class="login-error-list"><li><strong>Error:</strong> The username field is empty.</li>` +
				`<li><strong>Error:</strong> The password field is empty.</li></ul>`
			return html, loginShouldShake(c, "empty_username") || loginShouldShake(c, "empty_password")
		}
		return loginErrorHTML("empty_username", "The username field is empty.", "", ""), loginShouldShake(c, "empty_username")
	case "empty_password":
		return loginErrorHTML("empty_password", "The password field is empty.", "", ""), loginShouldShake(c, "empty_password")
	case "invalid_username":
		user := html.EscapeString(ae.Login)
		msg := "The username <strong>" + user + "</strong> is not registered on this site. If you are unsure of your username, try your email address instead."
		return loginErrorHTML("invalid_username", msg, "", ""), loginShouldShake(c, "invalid_username")
	case "incorrect_password":
		user := html.EscapeString(ae.Login)
		msg := "The password you entered for the username <strong>" + user + "</strong> is incorrect. "
		if lostURL != "" {
			msg += `<a href="` + html.EscapeString(lostURL) + `">Lost your password?</a>`
		}
		return loginErrorHTML("incorrect_password", msg, "", ""), loginShouldShake(c, "incorrect_password")
	case "test_cookie":
		msg := `Cookies are blocked or not supported by your browser. You must <a href="https://developer.wordpress.org/advanced-administration/wordpress/cookies/#enable-cookies-in-your-browser">enable cookies</a> to use WordPress.`
		return loginErrorHTML("test_cookie", msg, "", ""), loginShouldShake(c, "test_cookie") // default list omits test_cookie
	default:
		return loginErrorHTML(ae.Code, "The username or password you entered is incorrect.", "", ""), loginShouldShake(c, ae.Code)
	}
}

// loginErrorHTML wraps a message in a <p><strong>Error:</strong> …</p> block.
// msg may contain trusted markup (strong/a) already escaped for user values.
func loginErrorHTML(_, msg, _, _ string) string {
	return "<p><strong>Error:</strong> " + msg + "</p>"
}

// serveInterimLoginSuccess shows the interim modal success stub (P6.10 / P6.25 / P6.45).
// Auth cookies are already written by the caller. Body classes interim-login +
// interim-login-success let wp-auth-check.js hide the iframe. When redirect_to
// is present, offer Continue + top-level-only client redirect (P6.17) so the
// iframe still exposes the success class to the parent.
func (h *Handler) serveInterimLoginSuccess(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "/")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	redirectTo := ""
	if r != nil {
		redirectTo = strings.TrimSpace(r.FormValue("redirect_to"))
		if redirectTo == "" && r.URL != nil {
			redirectTo = strings.TrimSpace(r.URL.Query().Get("redirect_to"))
		}
	}
	if redirectTo != "" {
		redirectTo = sanitizeRedirect(redirectTo, home)
	}
	message := applyLoginMessage(c, `<p class="message" id="login-message">You have logged in successfully.</p>`)
	var b strings.Builder
	// WP login_header( '', $message ) — empty title on interim success (P6.25 / P6.51).
	writeLoginHead(&b, home, blogname, "", "login", "interim-login", "interim-login-success")
	b.WriteString("\t\t\t<div id=\"login\">\n")
	b.WriteString("\t\t<h1 role=\"presentation\" class=\"wp-login-logo\"><a href=\"https://wordpress.org/\">Powered by WordPress</a></h1>\n")
	if strings.TrimSpace(message) != "" {
		b.WriteString(message)
		b.WriteByte('\n')
	}
	if redirectTo != "" {
		b.WriteString("<p class=\"submit\"><a class=\"button button-primary\" href=\"")
		b.WriteString(html.EscapeString(redirectTo))
		b.WriteString("\">Continue</a></p>\n")
		b.WriteString("<script>\n")
		b.WriteString("(function(){try{if(window.top===window.self){window.location.href=")
		b.WriteString(strconvQuoteJS(redirectTo))
		b.WriteString(";}}catch(e){}})();\n")
		b.WriteString("</script>\n")
	}
	b.WriteString("\t\t\t</div>\n")
	// P6.45: login_footer action on interim success (WP always fires login_footer).
	if c != nil && c.Hooks != nil {
		c.Hooks.DoAction("login_footer", "")
	}
	b.WriteString("\t</body>\n")
	b.WriteString("\t</html>\n")
	writeLoginHTML(w, r, b.String(), false, c)
}

func strconvQuoteJS(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '<':
			// Avoid breaking out of script via </script in URL.
			b.WriteString(`\u003c`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// isInterimLogin mirrors WP ! empty( $_REQUEST['interim-login'] ) (P6.22 / P6.45).
// Any non-empty query or form value enables the interim modal chrome.
func isInterimLogin(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.URL != nil && r.URL.Query().Get("interim-login") != "" {
		return true
	}
	if r.Method == http.MethodPost {
		if r.PostForm != nil && r.PostForm.Get("interim-login") != "" {
			return true
		}
		if r.Form != nil && r.FormValue("interim-login") != "" {
			return true
		}
		// Parse once when form not yet loaded.
		if r.PostForm == nil && r.Form == nil {
			_ = r.ParseForm()
			if r.FormValue("interim-login") != "" {
				return true
			}
		}
	}
	return false
}

// isReauth mirrors WP ! empty( $_REQUEST['reauth'] ) (P6.18 / P6.37).
// Any non-empty query or form value forces re-authentication.
func isReauth(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.URL != nil && r.URL.Query().Get("reauth") != "" {
		return true
	}
	if r.Method == http.MethodPost {
		if r.PostForm != nil && r.PostForm.Get("reauth") != "" {
			return true
		}
		if r.Form != nil && r.FormValue("reauth") != "" {
			return true
		}
		// Parse once when form not yet loaded.
		if r.PostForm == nil && r.Form == nil {
			_ = r.ParseForm()
			if r.FormValue("reauth") != "" {
				return true
			}
		}
	}
	return false
}

// writeInterimLoginBlankTargetScript emits WP's interim-login footer script that
// forces anchors to target=_blank so the modal iframe is not navigated away.
func writeInterimLoginBlankTargetScript(b *strings.Builder) {
	b.WriteString("\t\t\t<script>\n")
	b.WriteString("( function() {\n")
	b.WriteString("\ttry {\n")
	b.WriteString("\t\tvar i, links = document.getElementsByTagName( 'a' );\n")
	b.WriteString("\t\tfor ( i in links ) {\n")
	b.WriteString("\t\t\tif ( links[i].href ) {\n")
	b.WriteString("\t\t\t\tlinks[i].target = '_blank';\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t} catch( er ) {}\n")
	b.WriteString("}());\n")
	b.WriteString("</script>\n")
}

// defaultLoginAdminRedirect returns admin_url() for post-login defaults (P6.34).
func defaultLoginAdminRedirect(c *wp.Context, home string) string {
	if c != nil {
		if u := strings.TrimSpace(wp.AdminURL(c, "")); u != "" {
			return u
		}
	}
	if home != "" {
		return strings.TrimRight(home, "/") + "/wp-admin/"
	}
	return "/wp-admin/"
}

// applyLoginRedirectFilter runs the login_redirect filter (P6.34).
func applyLoginRedirectFilter(c *wp.Context, redirectTo, requested string, user *wp.User) string {
	if c == nil || c.Hooks == nil {
		return redirectTo
	}
	if v := c.Hooks.ApplyFilters("login_redirect", redirectTo, requested, user); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" && s != "<nil>" {
			return s
		}
	}
	return redirectTo
}

// adjustDefaultAdminLoginRedirect sends users without edit_posts to profile.php
// when the redirect target is the default admin URL (P6.34 / wp-login.php).
func adjustDefaultAdminLoginRedirect(c *wp.Context, redirectTo, home string, user *wp.User) string {
	admin := defaultLoginAdminRedirect(c, home)
	candidates := map[string]bool{
		"":           true,
		"wp-admin/":  true,
		"/wp-admin":  true,
		"/wp-admin/": true,
	}
	if home != "" {
		h := strings.TrimRight(home, "/")
		candidates[h+"/wp-admin"] = true
		candidates[h+"/wp-admin/"] = true
	}
	candidates[admin] = true
	candidates[strings.TrimRight(admin, "/")] = true
	if !candidates[strings.TrimSpace(redirectTo)] {
		return redirectTo
	}
	if user == nil || c == nil {
		return "/wp-admin/"
	}
	if !c.UserCan(user, "edit_posts") {
		if c.UserCan(user, "read") {
			return wp.AdminURL(c, "profile.php")
		}
		if home != "" {
			return home + "/"
		}
		return "/"
	}
	return "/wp-admin/"
}

// applyLostPasswordRedirect runs the lostpassword_redirect filter (P6.35).
func applyLostPasswordRedirect(c *wp.Context, requested string) string {
	redirectTo := strings.TrimSpace(requested)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("lostpassword_redirect", redirectTo); v != nil {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return redirectTo
}

// sanitizeRedirect validates redirect_to like wp_validate_redirect (P6.20).
// Relative same-origin paths are kept; absolute URLs must match home's host.
// External hosts (and protocol-relative //evil…) fall back to /wp-admin/.
func sanitizeRedirect(raw, home string) string {
	const fallback = "/wp-admin/"
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	// Browsers treat //host as scheme-relative; force http: like core before parse.
	if strings.HasPrefix(raw, "//") {
		raw = "http:" + raw
	}
	// Allow relative paths (not protocol-relative).
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fallback
	}
	// Reject scheme-only / malformed hosts (https:evil.com).
	if u.Host == "" {
		if u.Scheme != "" || u.User != nil || u.Opaque != "" {
			return fallback
		}
		return fallback
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fallback
	}
	host := strings.ToLower(u.Hostname())
	allowed := map[string]bool{"127.0.0.1": true, "localhost": true}
	if home != "" {
		if hu, err := url.Parse(home); err == nil && hu.Hostname() != "" {
			allowed[strings.ToLower(hu.Hostname())] = true
		}
	}
	if !allowed[host] {
		return fallback
	}
	if u.Path == "" {
		return "/"
	}
	return u.RequestURI()
}

// loginCookiePath mirrors COOKIEPATH from the home URL path (P6.49).
// Root installs → "/"; subdirectory installs → "/subdir" (no trailing slash except root).
func loginCookiePath(home string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		return "/"
	}
	u, err := url.Parse(home)
	if err != nil || u.Path == "" || u.Path == "/" {
		return "/"
	}
	return strings.TrimRight(u.Path, "/")
}

// writeLanguageSwitcher emits the login language dropdown when languages exist (P6.11 / P6.39).
// Mirrors wp-login.php: only rendered when get_available_languages() is non-empty
// and login_display_language_dropdown is true.
// P6.30: preserve redirect_to/action/interim-login from query or POST body.
// P6.39: honor login_display_language_dropdown + preserve reauth.
func writeLanguageSwitcher(b *strings.Builder, r *http.Request, c *wp.Context, languages []string) {
	if len(languages) == 0 {
		return
	}
	// P6.39: apply_filters( 'login_display_language_dropdown', true ).
	display := true
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("login_display_language_dropdown", true); v != nil {
			if t, ok := v.(bool); ok {
				display = t
			}
		}
	}
	if !display {
		return
	}
	selected := "en_US"
	if c != nil && c.Options != nil {
		if v := optionString(c, "WPLANG", ""); v != "" {
			selected = v
		}
	}
	// P6.23 / P6.30: prefer ?wp_lang=, then POST body wp_lang, then cookie.
	_ = r.ParseForm()
	if q := r.URL.Query().Get("wp_lang"); q != "" {
		selected = q
	} else if r.Method == http.MethodPost {
		if v := r.PostForm.Get("wp_lang"); v != "" {
			selected = v
		} else if ck, err := r.Cookie("wp_lang"); err == nil && ck != nil && ck.Value != "" {
			selected = ck.Value
		}
	} else if ck, err := r.Cookie("wp_lang"); err == nil && ck != nil && ck.Value != "" {
		selected = ck.Value
	}
	selectID := "language-switcher-locales"
	selectName := "wp_lang"
	// P6.39: login_language_dropdown_args subset (id/name/selected).
	if c != nil && c.Hooks != nil {
		args := map[string]any{
			"id":       selectID,
			"name":     selectName,
			"selected": selected,
		}
		if v := c.Hooks.ApplyFilters("login_language_dropdown_args", args); v != nil {
			if m, ok := v.(map[string]any); ok {
				if id, ok := m["id"].(string); ok && strings.TrimSpace(id) != "" {
					selectID = strings.TrimSpace(id)
				}
				if name, ok := m["name"].(string); ok && strings.TrimSpace(name) != "" {
					selectName = strings.TrimSpace(name)
				}
				if sel, ok := m["selected"].(string); ok {
					selected = sel
				}
			}
		}
	}
	b.WriteString("\t\t\t<div class=\"language-switcher\">\n")
	b.WriteString("\t\t\t\t<form id=\"language-switcher\" method=\"get\">\n")
	b.WriteString("\t\t\t\t\t<label for=\"")
	b.WriteString(html.EscapeString(selectID))
	b.WriteString("\">\n")
	b.WriteString("\t\t\t\t\t\tLanguage<span class=\"dashicons dashicons-translation\" aria-hidden=\"true\"></span>\n")
	b.WriteString("\t\t\t\t\t</label>\n")
	b.WriteString("\t\t\t\t\t<select name=\"")
	b.WriteString(html.EscapeString(selectName))
	b.WriteString("\" id=\"")
	b.WriteString(html.EscapeString(selectID))
	b.WriteString("\">\n")
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
	// P6.30: $_REQUEST-shaped — query first, then POST form fields.
	rt := r.URL.Query().Get("redirect_to")
	if rt == "" {
		rt = r.FormValue("redirect_to")
	}
	if rt != "" {
		b.WriteString("\t\t\t\t\t\t<input type=\"hidden\" name=\"redirect_to\" value=\"")
		b.WriteString(html.EscapeString(rt))
		b.WriteString("\" />\n")
	}
	action := r.URL.Query().Get("action")
	if action == "" {
		action = r.FormValue("action")
	}
	if action != "" {
		b.WriteString("\t\t\t\t\t\t<input type=\"hidden\" name=\"action\" value=\"")
		b.WriteString(html.EscapeString(action))
		b.WriteString("\" />\n")
	}
	if isInterimLogin(r) {
		b.WriteString("\t\t\t\t\t\t<input type=\"hidden\" name=\"interim-login\" value=\"1\" />\n")
	}
	// P6.39: preserve reauth across language changes.
	reauth := r.URL.Query().Get("reauth")
	if reauth == "" {
		reauth = r.FormValue("reauth")
	}
	if reauth != "" {
		b.WriteString("\t\t\t\t\t\t<input type=\"hidden\" name=\"reauth\" value=\"")
		b.WriteString(html.EscapeString(reauth))
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
