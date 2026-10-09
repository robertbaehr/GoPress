package diffharness

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	testdiff "github.com/robertbaehr/GoPress/tests/diff"
)

var roleUsers = map[string]string{
	"admin":       "admin",
	"editor":      "editor",
	"author":      "author",
	"contributor": "contributor",
	"subscriber":  "subscriber",
}

// Target holds a base URL and per-role authenticated clients.
type Target struct {
	Name    string
	BaseURL string
	clients map[string]*http.Client
	nonces  map[string]map[string]string // role -> nonce-name -> value
}

func newTarget(name, base string) (*Target, error) {
	base = strings.TrimRight(base, "/")
	t := &Target{
		Name:    name,
		BaseURL: base,
		clients: map[string]*http.Client{},
		nonces:  map[string]map[string]string{},
	}
	// Anonymous client (no jar needed, but jar is fine).
	if err := t.ensureRole("anon"); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Target) ensureRole(role string) error {
	if _, ok := t.clients[role]; ok {
		return nil
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	client := &http.Client{
		Jar:     jar,
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	t.clients[role] = client
	t.nonces[role] = map[string]string{}
	if role == "anon" {
		return nil
	}
	user, ok := roleUsers[role]
	if !ok {
		return fmt.Errorf("unknown role %q", role)
	}
	return t.login(role, user, "password")
}

func (t *Target) login(role, user, pass string) error {
	client := t.clients[role]
	// Seed test cookie as wp-login expects.
	u, err := url.Parse(t.BaseURL + "/")
	if err != nil {
		return err
	}
	client.Jar.SetCookies(u, []*http.Cookie{{Name: "wordpress_test_cookie", Value: "WP Cookie check", Path: "/"}})

	form := url.Values{}
	form.Set("log", user)
	form.Set("pwd", pass)
	form.Set("wp-submit", "Log In")
	form.Set("redirect_to", t.BaseURL+"/wp-admin/")
	form.Set("testcookie", "1")

	req, err := http.NewRequest(http.MethodPost, t.BaseURL+"/wp-login.php", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("login %s as %s: status %d", t.Name, user, resp.StatusCode)
	}
	// Confirm logged-in cookie present.
	loggedIn := false
	for _, c := range client.Jar.Cookies(u) {
		if strings.HasPrefix(c.Name, "wordpress_logged_in_") {
			loggedIn = true
			break
		}
	}
	if !loggedIn {
		return fmt.Errorf("login %s as %s: no logged-in cookie (status %d, body %d bytes)", t.Name, user, resp.StatusCode, len(body))
	}
	return t.scrapeNonces(role)
}

var (
	wpAPINonceRe     = regexp.MustCompile(`wpApiSettings\s*=\s*\{[^}]*"nonce"\s*:\s*"([^"]+)"`)
	heartbeatNonceRe = regexp.MustCompile(`heartbeatSettings\s*=\s*\{[^}]*"nonce"\s*:\s*"([^"]+)"`)
	restNonceRe      = regexp.MustCompile(`(?i)"nonce"\s*:\s*"([a-zA-Z0-9]+)"`)
)

func (t *Target) scrapeNonces(role string) error {
	client := t.clients[role]
	req, err := http.NewRequest(http.MethodGet, t.BaseURL+"/wp-admin/", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	s := string(body)
	if m := wpAPINonceRe.FindStringSubmatch(s); len(m) == 2 {
		t.nonces[role]["wp_rest"] = m[1]
	} else if m := restNonceRe.FindStringSubmatch(s); len(m) == 2 {
		t.nonces[role]["wp_rest"] = m[1]
	}
	if m := heartbeatNonceRe.FindStringSubmatch(s); len(m) == 2 {
		t.nonces[role]["heartbeat"] = m[1]
	}
	return nil
}

func (t *Target) do(entry Entry) (testdiff.Response, error) {
	if err := t.ensureRole(entry.Role); err != nil {
		return testdiff.Response{}, err
	}
	client := t.clients[entry.Role]
	var bodyReader io.Reader
	bodyStr := ""
	if entry.Body != nil {
		bodyStr = *entry.Body
	}
	if entry.Nonce != nil && *entry.Nonce != "" {
		n := t.nonces[entry.Role][*entry.Nonce]
		if n == "" && *entry.Nonce != "wp_rest" {
			// Best-effort: scrape again.
			_ = t.scrapeNonces(entry.Role)
			n = t.nonces[entry.Role][*entry.Nonce]
		}
		if n == "" && *entry.Nonce == "wp_rest" {
			_ = t.scrapeNonces(entry.Role)
			n = t.nonces[entry.Role]["wp_rest"]
		}
		bodyStr = strings.ReplaceAll(bodyStr, "PLACEHOLDER", n)
		bodyStr = strings.ReplaceAll(bodyStr, "_nonce=PLACEHOLDER", "_nonce="+n)
	}
	if bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}
	req, err := http.NewRequest(strings.ToUpper(entry.Method), t.BaseURL+entry.Path, bodyReader)
	if err != nil {
		return testdiff.Response{}, err
	}
	for k, v := range entry.Headers {
		req.Header.Set(k, v)
	}
	if entry.Nonce != nil && *entry.Nonce == "wp_rest" {
		if n := t.nonces[entry.Role]["wp_rest"]; n != "" {
			req.Header.Set("X-WP-Nonce", n)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return testdiff.Response{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return testdiff.Response{}, err
	}
	return testdiff.Response{
		Status:  resp.StatusCode,
		Headers: resp.Header.Clone(),
		Body:    raw,
	}, nil
}
