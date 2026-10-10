// wp_die / die handlers (P2.95) — subset of wp-includes/functions.php.
// Go cannot exit the process like PHP die(); callers write DieResult to the response.
package wp

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
)

// DieHandler names mirror the wp_die_*_handler filter defaults.
const (
	DieHandlerDefault = "default" // _default_wp_die_handler
	DieHandlerAjax    = "ajax"    // _ajax_wp_die_handler
	DieHandlerJSON    = "json"    // _json_wp_die_handler
)

// DieArgs controls wp_die behavior (_wp_die_process_input defaults).
type DieArgs struct {
	Response         int
	Code             string
	Exit             bool // informational; Go handlers never os.Exit
	BackLink         bool
	LinkURL          string
	LinkText         string
	TextDirection    string
	Charset          string
	AdditionalErrors []DieExtraError
	ErrorData        any
}

// DieExtraError is one additional WP_Error entry surfaced in JSON die payloads.
type DieExtraError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// DieResult is the HTTP response wp_die would emit (without exiting).
type DieResult struct {
	Status      int
	ContentType string
	Body        []byte
	Handler     string
	Exit        bool
	Title       string
	Code        string
}

// SelectDieHandler chooses the die handler like wp_die() (ajax → json → default).
// Filterable via wp_die_ajax_handler / wp_die_json_handler / wp_die_handler.
func SelectDieHandler(c *Context, r *http.Request) string {
	handler := DieHandlerDefault
	filter := "wp_die_handler"
	if DoingAjax(c) {
		handler = DieHandlerAjax
		filter = "wp_die_ajax_handler"
	} else if IsJSONRequest(r) {
		handler = DieHandlerJSON
		filter = "wp_die_json_handler"
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters(filter, handler); v != nil {
			if s, ok := v.(string); ok && s != "" {
				return normalizeDieHandler(s)
			}
		}
	}
	return handler
}

func normalizeDieHandler(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case DieHandlerAjax, "_ajax_wp_die_handler":
		return DieHandlerAjax
	case DieHandlerJSON, "_json_wp_die_handler":
		return DieHandlerJSON
	case DieHandlerDefault, "_default_wp_die_handler":
		return DieHandlerDefault
	default:
		return name
	}
}

// ProcessDieInput mirrors _wp_die_process_input for string / WP_Error messages.
func ProcessDieInput(message any, title string, args DieArgs) (string, string, DieArgs) {
	if args.Charset == "" {
		args.Charset = "utf-8"
	}
	if args.TextDirection == "" {
		args.TextDirection = "ltr"
	}
	if args.TextDirection != "ltr" && args.TextDirection != "rtl" {
		args.TextDirection = "ltr"
	}
	args.Exit = true // PHP default; recorded on DieResult only

	msg := ""
	switch m := message.(type) {
	case nil:
		msg = ""
	case string:
		msg = m
	case *WPError:
		if m != nil && m.HasErrors() {
			code := m.GetErrorCode()
			msg = m.GetErrorMessage(code)
			if args.Code == "" {
				args.Code = code
			}
			if data := m.GetErrorData(code); data != nil {
				if dm, ok := data.(map[string]any); ok {
					if args.Response == 0 {
						if st, ok := asDieInt(dm["status"]); ok && st > 0 {
							args.Response = st
						}
					}
					if title == "" {
						if t, ok := dm["title"].(string); ok {
							title = t
						}
					}
				}
			}
			// Additional errors after the first.
			codes := m.GetErrorCodes()
			extras := make([]DieExtraError, 0)
			first := true
			for _, c := range codes {
				for _, em := range m.GetErrorMessages(c) {
					if first {
						first = false
						continue
					}
					extras = append(extras, DieExtraError{Code: c, Message: em, Data: m.GetErrorData(c)})
				}
			}
			args.AdditionalErrors = append(args.AdditionalErrors, extras...)
		}
	default:
		msg = fmt.Sprint(m)
	}

	if args.Code == "" {
		args.Code = "wp_die"
	}
	if args.Response == 0 {
		args.Response = 500
	}
	if title == "" {
		title = "WordPress \u203a Error"
	}
	return msg, title, args
}

// Die builds the response for wp_die(message, title, args) without exiting.
// Int title/args coercion matches PHP: wp_die($msg, 403) / wp_die($msg, $title, 403).
func Die(c *Context, r *http.Request, message any, title any, args any) DieResult {
	parsedTitle := ""
	parsedArgs := DieArgs{}
	switch t := title.(type) {
	case nil:
	case string:
		parsedTitle = t
	case int:
		parsedArgs.Response = t
	case int64:
		parsedArgs.Response = int(t)
	default:
		parsedTitle = fmt.Sprint(t)
	}
	switch a := args.(type) {
	case nil:
	case DieArgs:
		if parsedArgs.Response != 0 && a.Response == 0 {
			a.Response = parsedArgs.Response
		}
		parsedArgs = a
	case int:
		parsedArgs.Response = a
	case int64:
		parsedArgs.Response = int(a)
	case map[string]any:
		parsedArgs = dieArgsFromMap(a)
		if parsedArgs.Response == 0 {
			if resp, ok := title.(int); ok {
				parsedArgs.Response = resp
				parsedTitle = ""
			}
		}
	}

	handler := SelectDieHandler(c, r)
	// Ajax die defaults response to 200 before _wp_die_process_input (PHP).
	if handler == DieHandlerAjax && parsedArgs.Response == 0 {
		parsedArgs.Response = 200
	}
	msg, titleStr, parsedArgs := ProcessDieInput(message, parsedTitle, parsedArgs)
	switch handler {
	case DieHandlerAjax:
		return renderAjaxDie(msg, titleStr, parsedArgs)
	case DieHandlerJSON:
		return renderJSONDie(msg, titleStr, parsedArgs)
	default:
		return renderDefaultDie(msg, titleStr, parsedArgs)
	}
}

// WriteDie writes a DieResult to w (status, content-type, nocache, body).
func WriteDie(w http.ResponseWriter, res DieResult) {
	if w == nil {
		return
	}
	if res.ContentType != "" {
		w.Header().Set("Content-Type", res.ContentType)
	}
	NoCacheHeaders(w)
	status := res.Status
	if status == 0 {
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	if len(res.Body) > 0 {
		_, _ = w.Write(res.Body)
	}
}

func renderAjaxDie(message, title string, args DieArgs) DieResult {
	_ = title
	if args.Response == 0 {
		args.Response = 200
	}
	body := message
	if body == "" {
		body = "0"
	}
	return DieResult{
		Status:      args.Response,
		ContentType: "text/html; charset=" + args.Charset,
		Body:        []byte(body),
		Handler:     DieHandlerAjax,
		Exit:        args.Exit,
		Title:       title,
		Code:        args.Code,
	}
}

func renderJSONDie(message, title string, args DieArgs) DieResult {
	_ = title
	data := map[string]any{
		"code":    args.Code,
		"message": message,
		"data": map[string]any{
			"status": args.Response,
		},
		"additional_errors": args.AdditionalErrors,
	}
	if args.AdditionalErrors == nil {
		data["additional_errors"] = []DieExtraError{}
	}
	if args.ErrorData != nil {
		data["data"].(map[string]any)["error"] = args.ErrorData
	}
	raw, err := json.Marshal(data)
	if err != nil {
		raw = []byte(`{"code":"wp_die","message":"","data":{"status":500},"additional_errors":[]}`)
	}
	return DieResult{
		Status:      args.Response,
		ContentType: "application/json; charset=" + args.Charset,
		Body:        raw,
		Handler:     DieHandlerJSON,
		Exit:        args.Exit,
		Title:       title,
		Code:        args.Code,
	}
}

func renderDefaultDie(message, title string, args DieArgs) DieResult {
	msgHTML := message
	if len(args.AdditionalErrors) > 0 {
		var b strings.Builder
		b.WriteString("<ul>\n\t\t<li>")
		b.WriteString(message)
		for _, e := range args.AdditionalErrors {
			b.WriteString("</li>\n\t\t<li>")
			b.WriteString(e.Message)
		}
		b.WriteString("</li>\n\t</ul>")
		msgHTML = b.String()
	}
	msgHTML = `<div class="wp-die-message">` + msgHTML + `</div>`
	if args.LinkURL != "" && args.LinkText != "" {
		msgHTML += "\n<p><a href='" + html.EscapeString(args.LinkURL) + "'>" + html.EscapeString(args.LinkText) + "</a></p>"
	}
	if args.BackLink {
		msgHTML += "\n<p><a href='javascript:history.back()'>&laquo; Back</a></p>"
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html dir=\"")
	b.WriteString(html.EscapeString(args.TextDirection))
	b.WriteString("\">\n<head>\n")
	b.WriteString("<meta http-equiv=\"Content-Type\" content=\"text/html; charset=")
	b.WriteString(html.EscapeString(args.Charset))
	b.WriteString("\" />\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</title>\n</head>\n<body id=\"error-page\">\n")
	b.WriteString(msgHTML)
	b.WriteString("\n</body>\n</html>\n")
	return DieResult{
		Status:      args.Response,
		ContentType: "text/html; charset=" + args.Charset,
		Body:        []byte(b.String()),
		Handler:     DieHandlerDefault,
		Exit:        args.Exit,
		Title:       title,
		Code:        args.Code,
	}
}

func dieArgsFromMap(m map[string]any) DieArgs {
	a := DieArgs{}
	if v, ok := asDieInt(m["response"]); ok {
		a.Response = v
	}
	if s, ok := m["code"].(string); ok {
		a.Code = s
	}
	if s, ok := m["charset"].(string); ok {
		a.Charset = s
	}
	if s, ok := m["text_direction"].(string); ok {
		a.TextDirection = s
	}
	if s, ok := m["link_url"].(string); ok {
		a.LinkURL = s
	}
	if s, ok := m["link_text"].(string); ok {
		a.LinkText = s
	}
	if b, ok := m["back_link"].(bool); ok {
		a.BackLink = b
	}
	if b, ok := m["exit"].(bool); ok {
		a.Exit = b
	}
	return a
}

func asDieInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
}
