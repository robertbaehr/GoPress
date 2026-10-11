// HTTP client stub (P3.9) — wp_remote_get subset with WP_HTTP_BLOCK_EXTERNAL support.
package wp

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPResponse is a minimal wp_remote_get response.
type HTTPResponse struct {
	StatusCode int
	// Status is the reason phrase (response.message in WP), e.g. "OK" (P2.149).
	Status  string
	Body    string
	Headers http.Header
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
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, &HTTPError{Code: "http_request_failed", Message: err.Error()}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &HTTPError{Code: "http_request_failed", Message: err.Error()}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
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
	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Status:     status,
		Body:       string(body),
		Headers:    resp.Header.Clone(),
	}, nil
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
