// HTTP client stub (P3.9) — wp_remote_get subset with WP_HTTP_BLOCK_EXTERNAL support.
package wp

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPResponse is a minimal wp_remote_get response.
type HTTPResponse struct {
	StatusCode int
	Body       string
	Headers    http.Header
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
	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Headers:    resp.Header.Clone(),
	}, nil
}
