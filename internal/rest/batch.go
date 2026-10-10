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
	Body    any               `json:"body"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
}

// serveBatch handles POST /batch/v1 (WP REST batch controller stub, P4.35).
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

	allowed := map[string]bool{"POST": true, "PUT": true, "PATCH": true, "DELETE": true}
	responses := make([]batchSubResponse, 0, len(body.Requests))
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
		responses = append(responses, s.dispatchBatchSub(r, c, method, path, sub.Body))
	}
	writeJSON(w, http.StatusOK, map[string]any{"responses": responses})
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
	headers := map[string]string{}
	if allow := rec.Header().Get("Allow"); allow != "" {
		headers["Allow"] = allow
	}
	return batchSubResponse{
		Body:    parsed,
		Status:  rec.Code,
		Headers: headers,
	}
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
