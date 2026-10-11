// HTTP client stub (P3.9) — wp_remote_get subset with WP_HTTP_BLOCK_EXTERNAL support.
package wp

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPCookie is a minimal WP_Http_Cookie (name/value subset, P2.152).
type HTTPCookie struct {
	Name  string
	Value string
}

// HTTPResponse is a minimal wp_remote_get response.
type HTTPResponse struct {
	StatusCode int
	// Status is the reason phrase (response.message in WP), e.g. "OK" (P2.149).
	Status  string
	Body    string
	Headers http.Header
	// Cookies are Set-Cookie values from the response (wp_remote_retrieve_cookies, P2.152).
	Cookies []HTTPCookie
}

// HTTPError mirrors a WP_Error for blocked/failed HTTP requests.
type HTTPError struct {
	Code    string
	Message string
}

func (e *HTTPError) Error() string {
	if e == nil {
		return "wp: http error"
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// RemoteGet performs wp_remote_get.
// When Config.BlockExternal is true, returns http_request_not_executed without dialing.
func RemoteGet(c *Context, rawURL string) (*HTTPResponse, error) {
	return remoteRequest(c, http.MethodGet, rawURL, "")
}

// RemotePost performs wp_remote_post (P2.150).
// body is sent as the request body (WP $args['body'] string subset).
// When Config.BlockExternal is true, returns http_request_not_executed without dialing.
func RemotePost(c *Context, rawURL string, body string) (*HTTPResponse, error) {
	return remoteRequest(c, http.MethodPost, rawURL, body)
}

// RemoteHead performs wp_remote_head (P2.151).
// When Config.BlockExternal is true, returns http_request_not_executed without dialing.
func RemoteHead(c *Context, rawURL string) (*HTTPResponse, error) {
	return remoteRequest(c, http.MethodHead, rawURL, "")
}

// SafeRemoteGet mirrors wp_safe_remote_get (P2.147).
// Validates the URL with HttpValidateURL (reject_unsafe_urls) before RemoteGet.
func SafeRemoteGet(c *Context, rawURL string) (*HTTPResponse, error) {
	validated := HttpValidateURL(c, rawURL)
	if validated == "" {
		return nil, &HTTPError{
			Code:    "http_request_failed",
			Message: "A valid URL was not provided.",
		}
	}
	return RemoteGet(c, validated)
}

// SafeRemotePost mirrors wp_safe_remote_post (P2.150).
// Validates the URL with HttpValidateURL (reject_unsafe_urls) before RemotePost.
func SafeRemotePost(c *Context, rawURL string, body string) (*HTTPResponse, error) {
	validated := HttpValidateURL(c, rawURL)
	if validated == "" {
		return nil, &HTTPError{
			Code:    "http_request_failed",
			Message: "A valid URL was not provided.",
		}
	}
	return RemotePost(c, validated, body)
}

// SafeRemoteHead mirrors wp_safe_remote_head (P2.151).
// Validates the URL with HttpValidateURL (reject_unsafe_urls) before RemoteHead.
func SafeRemoteHead(c *Context, rawURL string) (*HTTPResponse, error) {
	validated := HttpValidateURL(c, rawURL)
	if validated == "" {
		return nil, &HTTPError{
			Code:    "http_request_failed",
			Message: "A valid URL was not provided.",
		}
	}
	return RemoteHead(c, validated)
}

// remoteRequest is the shared dial path for RemoteGet / RemotePost / RemoteHead.
func remoteRequest(c *Context, method, rawURL, body string) (*HTTPResponse, error) {
	if rawURL == "" {
		return nil, &HTTPError{Code: "http_request_failed", Message: "empty URL"}
	}
	if c != nil && c.Config != nil && c.Config.BlockExternal {
		return nil, &HTTPError{
			Code:    "http_request_not_executed",
			Message: "User has blocked requests through HTTP.",
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
	var bodyReader io.Reader
	if method != http.MethodGet && method != http.MethodHead {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		return nil, &HTTPError{Code: "http_request_failed", Message: err.Error()}
	}
	if method == http.MethodPost && body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &HTTPError{Code: "http_request_failed", Message: err.Error()}
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, &HTTPError{Code: "http_request_failed", Message: err.Error()}
	}
	status := strings.TrimSpace(resp.Status)
	if i := strings.IndexByte(status, ' '); i >= 0 && i+1 < len(status) {
		status = status[i+1:]
	}
	if status == "" {
		status = http.StatusText(resp.StatusCode)
	}
	cookies := make([]HTTPCookie, 0, len(resp.Cookies()))
	for _, ck := range resp.Cookies() {
		if ck == nil || ck.Name == "" {
			continue
		}
		cookies = append(cookies, HTTPCookie{Name: ck.Name, Value: ck.Value})
	}
	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Status:     status,
		Body:       string(respBody),
		Headers:    resp.Header.Clone(),
		Cookies:    cookies,
	}, nil
}

// RemoteRetrieveBody returns the response body string (wp_remote_retrieve_body, P2.148).
// Nil response or error-shaped values yield "".
func RemoteRetrieveBody(resp *HTTPResponse) string {
	if resp == nil {
		return ""
	}
	return resp.Body
}

// RemoteRetrieveResponseCode returns the HTTP status code (wp_remote_retrieve_response_code, P2.148 / P2.149).
// Nil response yields 0 (WP returns empty string for bad input; Go uses 0).
func RemoteRetrieveResponseCode(resp *HTTPResponse) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

// RemoteRetrieveResponseCodeOK reports whether the response code equals want (P2.149).
// Nil response is never OK.
func RemoteRetrieveResponseCodeOK(resp *HTTPResponse, want int) bool {
	if resp == nil {
		return false
	}
	return resp.StatusCode == want
}

// RemoteRetrieveResponseMessage returns the reason phrase (wp_remote_retrieve_response_message, P2.149).
// Nil response or empty Status yields "".
func RemoteRetrieveResponseMessage(resp *HTTPResponse) string {
	if resp == nil {
		return ""
	}
	if s := strings.TrimSpace(resp.Status); s != "" {
		return s
	}
	if resp.StatusCode > 0 {
		return http.StatusText(resp.StatusCode)
	}
	return ""
}

// RemoteRetrieveHeader returns a single header value (wp_remote_retrieve_header, P2.148).
// Missing header or nil response → "".
func RemoteRetrieveHeader(resp *HTTPResponse, header string) string {
	if resp == nil || resp.Headers == nil || strings.TrimSpace(header) == "" {
		return ""
	}
	return resp.Headers.Get(header)
}

// RemoteRetrieveHeaders returns the response header map (wp_remote_retrieve_headers, P2.148).
// Nil response → nil.
func RemoteRetrieveHeaders(resp *HTTPResponse) http.Header {
	if resp == nil {
		return nil
	}
	return resp.Headers
}

// RemoteRetrieveCookies returns cookies from the response (wp_remote_retrieve_cookies, P2.152).
// Nil response or no cookies → empty slice (never nil).
func RemoteRetrieveCookies(resp *HTTPResponse) []HTTPCookie {
	if resp == nil || len(resp.Cookies) == 0 {
		return []HTTPCookie{}
	}
	out := make([]HTTPCookie, len(resp.Cookies))
	copy(out, resp.Cookies)
	return out
}

// RemoteRetrieveCookie returns the first cookie matching name (wp_remote_retrieve_cookie, P2.152).
// Missing cookie or nil response → zero HTTPCookie and ok=false (WP returns ”).
func RemoteRetrieveCookie(resp *HTTPResponse, name string) (HTTPCookie, bool) {
	name = strings.TrimSpace(name)
	if resp == nil || name == "" {
		return HTTPCookie{}, false
	}
	for _, ck := range resp.Cookies {
		if ck.Name == name {
			return ck, true
		}
	}
	return HTTPCookie{}, false
}

// RemoteRetrieveCookieValue returns a cookie value by name (wp_remote_retrieve_cookie_value, P2.152).
// Missing cookie or nil response → "".
func RemoteRetrieveCookieValue(resp *HTTPResponse, name string) string {
	ck, ok := RemoteRetrieveCookie(resp, name)
	if !ok {
		return ""
	}
	return ck.Value
}
