package blocks

import (
	"html"
	"net/url"
	"strings"
)

// renderLoginout renders core/loginout (P5.74 / P5.121) — Log in / Log out link,
// or an optional login form when displayLoginAsForm is set for logged-out visitors.
// P5.121: className / has-link-color / backgroundColor / anchor like
// get_block_wrapper_attributes; form submit gets wp-block-button__link.
func (e *Env) renderLoginout(block Block) string {
	e = e.ensure()
	loggedIn := e.IsLoggedIn != nil && e.IsLoggedIn()
	redirect := ""
	if redirectToCurrent(block.Attrs) {
		if e.CurrentURL != nil {
			redirect = strings.TrimSpace(e.CurrentURL())
		}
	}

	classes := []string{"wp-block-loginout"}
	if loggedIn {
		classes = append(classes, "logged-in")
	} else {
		classes = append(classes, "logged-out")
	}

	var contents string
	if !loggedIn && displayLoginAsForm(block.Attrs) {
		classes = append(classes, "has-login-form")
		contents = e.renderLoginForm(redirect)
	} else {
		contents = e.renderLoginoutLink(loggedIn, redirect)
	}

	// P5.121: wrapper attrs subset of get_block_wrapper_attributes.
	if hasLinkColorAttr(block.Attrs) {
		classes = append(classes, "has-link-color")
	}
	if bg := strings.TrimSpace(attrString(block.Attrs, "backgroundColor")); bg != "" {
		classes = append(classes, "has-background", "has-"+bg+"-background-color")
	}
	if hasCustomBackgroundColor(block.Attrs) {
		classes = append(classes, "has-background")
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}

	attrs := `class="` + html.EscapeString(strings.Join(classes, " ")) + `"`
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		attrs += ` id="` + html.EscapeString(anchor) + `"`
	}

	return `<div ` + attrs + `>` + contents + `</div>`
}

func (e *Env) renderLoginoutLink(loggedIn bool, redirect string) string {
	e = e.ensure()
	if loggedIn {
		href := "/wp-login.php?action=logout"
		if e.LogoutURL != nil {
			href = e.LogoutURL(redirect)
		} else if redirect != "" {
			href += "&redirect_to=" + url.QueryEscape(redirect)
		}
		return `<a href="` + html.EscapeString(href) + `">Log out</a>`
	}
	href := "/wp-login.php"
	if e.LoginURL != nil {
		href = e.LoginURL(redirect)
	} else if redirect != "" {
		href += "?redirect_to=" + url.QueryEscape(redirect)
	}
	return `<a href="` + html.EscapeString(href) + `">Log in</a>`
}

// renderLoginForm is a minimal wp_login_form subset for displayLoginAsForm (P5.74 / P6.52).
func (e *Env) renderLoginForm(redirect string) string {
	e = e.ensure()
	action := "/wp-login.php"
	if e.LoginURL != nil {
		action = e.LoginURL("")
	}
	if redirect == "" && e.CurrentURL != nil {
		redirect = strings.TrimSpace(e.CurrentURL())
	}
	args := map[string]any{
		"echo":           true,
		"redirect":       redirect,
		"form_id":        "loginform",
		"label_username": "Username or Email Address",
		"label_password": "Password",
		"label_remember": "Remember Me",
		"label_log_in":   "Log In",
		"id_username":    "user_login",
		"id_password":    "user_pass",
		"id_remember":    "rememberme",
		"id_submit":      "wp-submit",
		"remember":       true,
		"value_username": "",
		"value_remember": false,
	}
	// P6.52: login_form_top / middle / bottom filters (wp_login_form).
	top := e.applyLoginFormFilter("login_form_top", "", args)
	middle := e.applyLoginFormFilter("login_form_middle", "", args)
	bottom := e.applyLoginFormFilter("login_form_bottom", "", args)

	var b strings.Builder
	b.WriteString(`<form name="loginform" id="loginform" action="`)
	b.WriteString(html.EscapeString(action))
	b.WriteString(`" method="post">`)
	b.WriteString(top)
	b.WriteString(`<p class="login-username"><label for="user_login">Username or Email Address</label>`)
	b.WriteString(`<input type="text" name="log" id="user_login" autocomplete="username" class="input" value="" size="20" /></p>`)
	b.WriteString(`<p class="login-password"><label for="user_pass">Password</label>`)
	b.WriteString(`<input type="password" name="pwd" id="user_pass" autocomplete="current-password" spellcheck="false" class="input" value="" size="20" /></p>`)
	b.WriteString(middle)
	b.WriteString(`<p class="login-remember"><label><input name="rememberme" type="checkbox" id="rememberme" value="forever" /> Remember Me</label></p>`)
	// P5.121: block themes add wp-block-button__link on the submit input.
	b.WriteString(`<p class="login-submit"><input type="submit" name="wp-submit" id="wp-submit" class="button button-primary wp-block-button__link" value="Log In" />`)
	b.WriteString(`<input type="hidden" name="redirect_to" value="`)
	b.WriteString(html.EscapeString(redirect))
	b.WriteString(`" /></p>`)
	b.WriteString(bottom)
	b.WriteString(`</form>`)
	return b.String()
}

func (e *Env) applyLoginFormFilter(tag, value string, args map[string]any) string {
	e = e.ensure()
	if e.ApplyFilter == nil {
		return value
	}
	filtered := e.ApplyFilter(tag, value, args)
	if s, ok := filtered.(string); ok {
		return s
	}
	return value
}

func redirectToCurrent(attrs map[string]any) bool {
	if attrs == nil {
		return true // block.json default
	}
	v, ok := attrs["redirectToCurrent"]
	if !ok || v == nil {
		return true
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "false", "0", "no", "off":
			return false
		default:
			return true
		}
	default:
		return true
	}
}

func displayLoginAsForm(attrs map[string]any) bool {
	if attrs == nil {
		return false
	}
	v, ok := attrs["displayLoginAsForm"]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "true", "yes", "on":
			return true
		default:
			return false
		}
	default:
		return false
	}
}
