package wp

// GetSessionToken returns the current session token from the logged_in
// auth cookie (wp_get_session_token).
//
// Prefer Context.SessionToken when AuthenticateRequest already populated it;
// otherwise parse the logged_in cookie on the request. Returns "" when
// anonymous or the cookie is missing/malformed.
func GetSessionToken(c *Context) string {
	if c == nil {
		return ""
	}
	if t := c.SessionToken; t != "" {
		return t
	}
	if c.Req == nil {
		return ""
	}
	name := c.AuthCookieName("logged_in")
	cookie, err := c.Req.Cookie(name)
	if err != nil || cookie == nil || cookie.Value == "" {
		return ""
	}
	parts, err := ParseAuthCookie(cookie.Value, "logged_in")
	if err != nil || parts == nil {
		return ""
	}
	return parts.Token
}
