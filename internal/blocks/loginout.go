package blocks

import (
	"html"
	"net/url"
	"strings"
)

// renderLoginout renders core/loginout (P5.74) — Log in / Log out link, or an
// optional login form when displayLoginAsForm is set for logged-out visitors.
func (e *Env) renderLoginout(block Block) string {
	e = e.ensure()
	loggedIn := e.IsLoggedIn != nil && e.IsLoggedIn()
	redirect := ""
	if redirectToCurrent(block.Attrs) {
		if e.CurrentURL != nil {
			redirect = strings.TrimSpace(e.CurrentURL())
		}
	}

	classes := "wp-block-loginout"
	if loggedIn {
		classes += " logged-in"
	} else {
		classes += " logged-out"
	}

	var contents string
	if !loggedIn && displayLoginAsForm(block.Attrs) {
		classes += " has-login-form"
		contents = e.renderLoginForm(redirect)
	} else {
		contents = e.renderLoginoutLink(loggedIn, redirect)
	}

	return `<div class="` + html.EscapeString(classes) + `">` + contents + `</div>`
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

// renderLoginForm is a minimal wp_login_form subset for displayLoginAsForm.
func (e *Env) renderLoginForm(redirect string) string {
	e = e.ensure()
	action := "/wp-login.php"
	if e.LoginURL != nil {
		action = e.LoginURL("")
	}
	if redirect == "" && e.CurrentURL != nil {
		redirect = strings.TrimSpace(e.CurrentURL())
	}
	var b strings.Builder
	b.WriteString(`<form name="loginform" id="loginform" action="`)
	b.WriteString(html.EscapeString(action))
	b.WriteString(`" method="post">`)
	b.WriteString(`<p class="login-username"><label for="user_login">Username or Email Address</label>`)
	b.WriteString(`<input type="text" name="log" id="user_login" autocomplete="username" class="input" value="" size="20" /></p>`)
	b.WriteString(`<p class="login-password"><label for="user_pass">Password</label>`)
	b.WriteString(`<input type="password" name="pwd" id="user_pass" autocomplete="current-password" spellcheck="false" class="input" value="" size="20" /></p>`)
	b.WriteString(`<p class="login-remember"><label><input name="rememberme" type="checkbox" id="rememberme" value="forever" /> Remember Me</label></p>`)
	b.WriteString(`<p class="login-submit"><input type="submit" name="wp-submit" id="wp-submit" class="button button-primary" value="Log In" />`)
	b.WriteString(`<input type="hidden" name="redirect_to" value="`)
	b.WriteString(html.EscapeString(redirect))
	b.WriteString(`" /></p></form>`)
	return b.String()
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
