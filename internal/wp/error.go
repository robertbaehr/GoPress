package wp

// WPError mirrors WP_Error (P2.93) — container for error codes/messages/data.
type WPError struct {
	Errors         map[string][]string
	ErrorData      map[string]any
	additionalData map[string][]any
	codeOrder      []string // insertion order for get_error_code / get_error_codes
}

// NewWPError constructs a WP_Error. Empty code yields an empty error object.
func NewWPError(code, message string, data any) *WPError {
	e := &WPError{
		Errors:         map[string][]string{},
		ErrorData:      map[string]any{},
		additionalData: map[string][]any{},
	}
	if code == "" {
		return e
	}
	e.Add(code, message, data)
	return e
}

// Add appends a message for code (WP_Error::add) and optional data.
func (e *WPError) Add(code, message string, data any) {
	e.AddWithHooks(nil, code, message, data)
}

// AddWithHooks is WP_Error::add and fires wp_error_added when hooks are set (P2.94).
func (e *WPError) AddWithHooks(h *Hooks, code, message string, data any) {
	if e == nil || code == "" {
		return
	}
	if e.Errors == nil {
		e.Errors = map[string][]string{}
	}
	if _, exists := e.Errors[code]; !exists {
		e.codeOrder = append(e.codeOrder, code)
	}
	e.Errors[code] = append(e.Errors[code], message)
	if data != nil && data != "" {
		e.AddData(data, code)
	}
	if h != nil {
		h.DoAction("wp_error_added", code, message, data, e)
	}
}

// AddData stores error data for code (WP_Error::add_data).
func (e *WPError) AddData(data any, code string) {
	if e == nil {
		return
	}
	if code == "" {
		code = e.GetErrorCode()
	}
	if code == "" {
		return
	}
	if e.ErrorData == nil {
		e.ErrorData = map[string]any{}
	}
	if e.additionalData == nil {
		e.additionalData = map[string][]any{}
	}
	if prev, ok := e.ErrorData[code]; ok {
		e.additionalData[code] = append(e.additionalData[code], prev)
	}
	e.ErrorData[code] = data
}

// HasErrors reports whether any error codes are present (WP_Error::has_errors).
func (e *WPError) HasErrors() bool {
	return e != nil && len(e.Errors) > 0
}

// GetErrorCodes returns all error codes (WP_Error::get_error_codes).
func (e *WPError) GetErrorCodes() []string {
	if !e.HasErrors() {
		return []string{}
	}
	out := make([]string, 0, len(e.codeOrder))
	for _, code := range e.codeOrder {
		if _, ok := e.Errors[code]; ok {
			out = append(out, code)
		}
	}
	return out
}

// GetErrorCode returns the first error code (WP_Error::get_error_code).
func (e *WPError) GetErrorCode() string {
	codes := e.GetErrorCodes()
	if len(codes) == 0 {
		return ""
	}
	return codes[0]
}

// GetErrorMessages returns messages for code, or all when code is empty.
func (e *WPError) GetErrorMessages(code string) []string {
	if e == nil || e.Errors == nil {
		return []string{}
	}
	if code == "" {
		var all []string
		for _, msgs := range e.Errors {
			all = append(all, msgs...)
		}
		return all
	}
	if msgs, ok := e.Errors[code]; ok {
		return append([]string{}, msgs...)
	}
	return []string{}
}

// GetErrorMessage returns the first message for code (or first code).
func (e *WPError) GetErrorMessage(code string) string {
	if code == "" {
		code = e.GetErrorCode()
	}
	msgs := e.GetErrorMessages(code)
	if len(msgs) == 0 {
		return ""
	}
	return msgs[0]
}

// GetErrorData returns the most recent data for code.
func (e *WPError) GetErrorData(code string) any {
	if e == nil || e.ErrorData == nil {
		return nil
	}
	if code == "" {
		code = e.GetErrorCode()
	}
	return e.ErrorData[code]
}

// GetAllErrorData returns all data for code in insertion order (WP_Error::get_all_error_data, P2.94).
func (e *WPError) GetAllErrorData(code string) []any {
	if e == nil {
		return []any{}
	}
	if code == "" {
		code = e.GetErrorCode()
	}
	if code == "" {
		return []any{}
	}
	var data []any
	if e.additionalData != nil {
		if prev, ok := e.additionalData[code]; ok {
			data = append(data, prev...)
		}
	}
	if e.ErrorData != nil {
		if latest, ok := e.ErrorData[code]; ok {
			data = append(data, latest)
		}
	}
	return data
}

// MergeFrom merges errors from other into e (WP_Error::merge_from, P2.94).
func (e *WPError) MergeFrom(other *WPError) {
	copyWPErrors(other, e)
}

// ExportTo exports errors from e into other (WP_Error::export_to, P2.94).
func (e *WPError) ExportTo(other *WPError) {
	copyWPErrors(e, other)
}

func copyWPErrors(from, to *WPError) {
	if from == nil || to == nil || !from.HasErrors() {
		return
	}
	for _, code := range from.GetErrorCodes() {
		for _, msg := range from.GetErrorMessages(code) {
			to.Add(code, msg, nil)
		}
		for _, data := range from.GetAllErrorData(code) {
			to.AddData(data, code)
		}
	}
}

// Remove clears messages and data for code (WP_Error::remove).
func (e *WPError) Remove(code string) {
	if e == nil || code == "" {
		return
	}
	delete(e.Errors, code)
	delete(e.ErrorData, code)
	delete(e.additionalData, code)
	if len(e.codeOrder) > 0 {
		next := e.codeOrder[:0]
		for _, c := range e.codeOrder {
			if c != code {
				next = append(next, c)
			}
		}
		e.codeOrder = next
	}
}

// Error implements the Go error interface using the first message.
func (e *WPError) Error() string {
	if e == nil {
		return ""
	}
	msg := e.GetErrorMessage("")
	if msg == "" {
		if code := e.GetErrorCode(); code != "" {
			return code
		}
		return "wp_error"
	}
	return msg
}

// IsWPError reports whether thing is a *WPError (is_wp_error).
func IsWPError(thing any) bool {
	return IsWPErrorWithHooks(nil, thing)
}

// IsWPErrorWithHooks is is_wp_error with optional is_wp_error_instance action.
func IsWPErrorWithHooks(h *Hooks, thing any) bool {
	err, ok := thing.(*WPError)
	if !ok || err == nil {
		return false
	}
	if h != nil {
		h.DoAction("is_wp_error_instance", err)
	}
	return true
}

// IsWPErrorContext is is_wp_error using the context hooks when available.
func IsWPErrorContext(c *Context, thing any) bool {
	var h *Hooks
	if c != nil {
		h = c.Hooks
	}
	return IsWPErrorWithHooks(h, thing)
}
