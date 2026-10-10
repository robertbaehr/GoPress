package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

type batchRequestBody struct {
	Validation string            `json:"validation"`
	Requests   []batchSubRequest `json:"requests"`
}

type batchSubRequest struct {
	Method  string         `json:"method"`
	Path    string         `json:"path"`
	Body    map[string]any `json:"body"`
	Headers map[string]any `json:"headers"`
}

type batchSubResponse struct {
	Body    any `json:"body"`
	Status  int `json:"status"`
	Headers any `json:"headers"`
}

// serveBatch handles POST /batch/v1 (WP REST batch controller, P4.35 / P4.70).
func (s *Server) serveBatch(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": "Invalid JSON body.",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	var body batchRequestBody
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_json",
				"message": "Invalid JSON body.",
				"data":    map[string]any{"status": 400},
			})
			return
		}
	}
	if body.Requests == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_missing_callback_param",
			"message": "Missing parameter(s): requests",
			"data":    map[string]any{"status": 400, "params": []string{"requests"}},
		})
		return
	}
	if len(body.Requests) > 25 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid parameter(s): requests",
			"data": map[string]any{
				"status": 400,
				"params": map[string]any{"requests": "requests must contain at most 25 items."},
			},
		})
		return
	}

	validation := strings.TrimSpace(body.Validation)
	if validation == "" {
		validation = "normal"
	}
	if validation != "normal" && validation != "require-all-validate" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid parameter(s): validation",
			"data": map[string]any{
				"status": 400,
				"params": map[string]any{"validation": "validation is not one of require-all-validate and normal."},
			},
		})
		return
	}

	allowed := map[string]bool{"POST": true, "PUT": true, "PATCH": true, "DELETE": true}
	type prepared struct {
		method string
		path   string
		body   map[string]any
	}
	prepped := make([]prepared, 0, len(body.Requests))
	for i, sub := range body.Requests {
		method := strings.ToUpper(strings.TrimSpace(sub.Method))
		if method == "" {
			method = "POST"
		}
		if !allowed[method] {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": "Invalid parameter(s): requests",
				"data": map[string]any{
					"status": 400,
					"params": map[string]any{
						"requests": "requests[" + itoa(i) + "][method] is not one of POST, PUT, PATCH, and DELETE.",
					},
					"details": map[string]any{
						"requests": map[string]any{
							"code":    "rest_not_in_enum",
							"message": "requests[" + itoa(i) + "][method] is not one of POST, PUT, PATCH, and DELETE.",
							"data":    nil,
						},
					},
				},
			})
			return
		}
		path := strings.TrimSpace(sub.Path)
		if path == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": "Invalid parameter(s): requests",
				"data": map[string]any{
					"status": 400,
					"params": map[string]any{"requests": "requests[" + itoa(i) + "][path] is required."},
				},
			})
			return
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		prepped = append(prepped, prepared{method: method, path: path, body: sub.Body})
	}

	dispatched := make([]batchSubResponse, 0, len(prepped))
	for _, sub := range prepped {
		dispatched = append(dispatched, s.dispatchBatchSub(r, c, sub.method, sub.path, sub.body))
	}

	if validation == "require-all-validate" {
		hasError := false
		for _, resp := range dispatched {
			if resp.Status >= 400 {
				hasError = true
				break
			}
		}
		if hasError {
			// Mirror WP: null for ok slots; keep error envelopes for failures.
			responses := make([]any, 0, len(dispatched))
			for _, resp := range dispatched {
				if resp.Status >= 400 {
					responses = append(responses, resp)
				} else {
					responses = append(responses, nil)
				}
			}
			writeJSON(w, http.StatusMultiStatus, map[string]any{
				"failed":    "validation",
				"responses": responses,
			})
			return
		}
	}

	// WP returns 207 Multi-Status for batch/v1 (P4.70).
	writeJSON(w, http.StatusMultiStatus, map[string]any{"responses": dispatched})
}

func (s *Server) dispatchBatchSub(parent *http.Request, c *wp.Context, method, path string, body map[string]any) batchSubResponse {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	} else {
		payload = []byte("{}")
	}
	url := "/wp-json" + path
	req := httptest.NewRequest(method, url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	// Carry auth cookies from the parent batch request.
	if parent != nil {
		for _, ck := range parent.Cookies() {
			req.AddCookie(ck)
		}
		if auth := parent.Header.Get("Authorization"); auth != "" {
			req.Header.Set("Authorization", auth)
		}
	}
	rec := httptest.NewRecorder()
	s.Handle(rec, req, c)
	var parsed any
	_ = json.Unmarshal(rec.Body.Bytes(), &parsed)
	return batchSubResponse{
		Body:    parsed,
		Status:  rec.Code,
		Headers: batchResponseHeaders(rec),
	}
}

// batchResponseHeaders copies WP-relevant response headers into an envelope map.
// Empty headers encode as [] to match PHP's empty associative array JSON.
func batchResponseHeaders(rec *httptest.ResponseRecorder) any {
	headers := map[string]string{}
	if allow := rec.Header().Get("Allow"); allow != "" {
		headers["Allow"] = allow
	}
	if link := rec.Header().Get("Link"); link != "" {
		headers["Link"] = link
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		headers["Location"] = loc
	}
	if xTotal := rec.Header().Get("X-WP-Total"); xTotal != "" {
		headers["X-WP-Total"] = xTotal
	}
	if xPages := rec.Header().Get("X-WP-TotalPages"); xPages != "" {
		headers["X-WP-TotalPages"] = xPages
	}
	if len(headers) == 0 {
		return []any{}
	}
	return headers
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [12]byte
	pos := len(b)
	n := i
	for n > 0 {
		pos--
		b[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(b[pos:])
}

func batchRouteIndex() RouteIndex {
	return RouteIndex{
		Namespace: "",
		Methods:   []string{"POST"},
		Endpoints: []RouteEndpoint{{
			Methods: []string{"POST"},
			Args: map[string]any{
				"validation": map[string]any{
					"type":     "string",
					"enum":     []string{"require-all-validate", "normal"},
					"default":  "normal",
					"required": false,
				},
				"requests": map[string]any{
					"type":     "array",
					"maxItems": 25,
					"required": true,
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"method": map[string]any{
								"type":    "string",
								"enum":    []string{"POST", "PUT", "PATCH", "DELETE"},
								"default": "POST",
							},
							"path": map[string]any{
								"type":     "string",
								"required": true,
							},
							"body": map[string]any{
								"type":                 "object",
								"properties":           []any{},
								"additionalProperties": true,
							},
							"headers": map[string]any{
								"type":       "object",
								"properties": []any{},
								"additionalProperties": map[string]any{
									"type":  []string{"string", "array"},
									"items": map[string]any{"type": "string"},
								},
							},
						},
					},
				},
			},
		}},
	}
}
