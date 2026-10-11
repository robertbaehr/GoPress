package wp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// UserRequestKeyExpiration is DAY_IN_SECONDS (user_request_key_expiration default).
const UserRequestKeyExpiration = 24 * 60 * 60

// UserRequest is a privacy user_request post stub (P8.21).
type UserRequest struct {
	ID          int64
	Email       string
	ActionName  string // export_personal_data | remove_personal_data
	Status      string // request-pending | request-confirmed | request-completed
	Requested   time.Time
	Description string
}

// CreateUserRequest mirrors wp_create_user_request (P8.21).
// status is "pending" or "confirmed" (stored as request-pending / request-confirmed).
func CreateUserRequest(c *Context, email, actionName, status string) (*UserRequest, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	actionName = strings.TrimSpace(actionName)
	status = strings.TrimSpace(status)
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("wp: invalid_email")
	}
	switch actionName {
	case "export_personal_data", "remove_personal_data":
	default:
		return nil, fmt.Errorf("wp: invalid_action")
	}
	switch status {
	case "pending", "confirmed":
	default:
		return nil, fmt.Errorf("wp: invalid_status")
	}
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: CreateUserRequest: invalid state")
	}

	// Duplicate incomplete request for same email+action.
	existing, err := ListUserRequests(c, actionName, 50)
	if err != nil {
		return nil, err
	}
	for _, r := range existing {
		if strings.EqualFold(r.Email, email) &&
			(r.Status == "request-pending" || r.Status == "request-confirmed") {
			return nil, fmt.Errorf("wp: duplicate_request")
		}
	}

	author := int64(0)
	if u, uerr := c.GetUserBy("email", email); uerr == nil && u != nil {
		author = u.ID
	}
	now := time.Now().UTC()
	stamp := now.Format("2006-01-02 15:04:05")
	postStatus := "request-" + status
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		author, stamp, stamp, "{}", email, "", postStatus,
		"closed", "closed", "", actionName, "", "", stamp, stamp,
		"", 0, "", 0, "user_request", "", 0,
	)
	if err != nil {
		return nil, err
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &UserRequest{
		ID:          id,
		Email:       email,
		ActionName:  actionName,
		Status:      postStatus,
		Requested:   now,
		Description: UserRequestActionDescription(actionName),
	}, nil
}

// UserRequestActionDescription mirrors wp_user_request_action_description (P8.21).
func UserRequestActionDescription(actionName string) string {
	switch actionName {
	case "export_personal_data":
		return "Export Personal Data"
	case "remove_personal_data":
		return "Erase Personal Data"
	default:
		return "Confirm the \"" + actionName + "\" action"
	}
}

// ListUserRequests returns recent user_request posts for an action (P8.21).
func ListUserRequests(c *Context, actionName string, limit int) ([]UserRequest, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: ListUserRequests: invalid state")
	}
	if limit <= 0 {
		limit = 20
	}
	actionName = strings.TrimSpace(actionName)
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_title, post_name, post_status, post_date_gmt, post_date FROM "+c.DB.Posts()+
			" WHERE post_type = %s AND post_name = %s ORDER BY ID DESC LIMIT %d",
		"user_request", actionName, limit,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]UserRequest, 0, len(rows))
	for _, row := range rows {
		stamp := rowString(row["post_date_gmt"])
		if stamp == "" {
			stamp = rowString(row["post_date"])
		}
		var when time.Time
		if t, err := time.Parse("2006-01-02 15:04:05", stamp); err == nil {
			when = t.UTC()
		}
		action := rowString(row["post_name"])
		out = append(out, UserRequest{
			ID:          rowInt64(row["ID"]),
			Email:       rowString(row["post_title"]),
			ActionName:  action,
			Status:      rowString(row["post_status"]),
			Requested:   when,
			Description: UserRequestActionDescription(action),
		})
	}
	return out, nil
}

// CompleteUserRequest marks a user_request completed (P8.21 stub).
func CompleteUserRequest(c *Context, id int64) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("wp: CompleteUserRequest: invalid state")
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET post_status = %s, post_modified = %s, post_modified_gmt = %s "+
			"WHERE ID = %d AND post_type = %s",
		"request-completed", now, now, id, "user_request",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

// FindOpenUserRequestID returns the newest pending/confirmed request ID for email+action.
func FindOpenUserRequestID(c *Context, email, actionName string) int64 {
	rows, err := ListUserRequests(c, actionName, 50)
	if err != nil {
		return 0
	}
	email = strings.TrimSpace(strings.ToLower(email))
	for _, r := range rows {
		if strings.EqualFold(r.Email, email) &&
			(r.Status == "request-pending" || r.Status == "request-confirmed") {
			return r.ID
		}
	}
	return 0
}

// GetUserRequest loads a user_request by ID (wp_get_user_request subset, P6.46).
func GetUserRequest(c *Context, requestID int64) (*UserRequest, error) {
	if c == nil || c.DB == nil || requestID <= 0 {
		return nil, fmt.Errorf("wp: GetUserRequest: invalid state")
	}
	post, err := GetPostByID(c, requestID)
	if err != nil || post == nil || post.Type != "user_request" {
		return nil, fmt.Errorf("wp: invalid_request")
	}
	when := time.Time{}
	if t, err := time.Parse("2006-01-02 15:04:05", post.Date); err == nil {
		when = t.UTC()
	}
	return &UserRequest{
		ID:          post.ID,
		Email:       post.Title,
		ActionName:  post.Name,
		Status:      post.Status,
		Requested:   when,
		Description: UserRequestActionDescription(post.Name),
	}, nil
}

// GenerateUserRequestKey mirrors wp_generate_user_request_key (P6.46).
// Stores wp_fast_hash(key) in post_password and resets status to request-pending.
func GenerateUserRequestKey(c *Context, requestID int64) (string, error) {
	if c == nil || c.DB == nil || requestID <= 0 {
		return "", fmt.Errorf("wp: GenerateUserRequestKey: invalid state")
	}
	key := GeneratePassword(20, false, false)
	hash := FastHash(key)
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET post_password = %s, post_status = %s, post_modified = %s, post_modified_gmt = %s "+
			"WHERE ID = %d AND post_type = %s",
		hash, "request-pending", now, now, requestID, "user_request",
	)
	if err != nil {
		return "", err
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		return "", err
	}
	return key, nil
}

// ValidateUserRequestKey mirrors wp_validate_user_request_key (P6.46).
// Returns nil on success, or a *WPError on failure.
func ValidateUserRequestKey(c *Context, requestID int64, key string) error {
	requestID = AbsInt(requestID)
	key = strings.TrimSpace(key)
	post, err := GetPostByID(c, requestID)
	if err != nil || post == nil || post.Type != "user_request" {
		return NewWPError("invalid_request", "Invalid personal data request.", map[string]any{"status": 500})
	}
	savedKey := strings.TrimSpace(post.Password)
	if savedKey == "" {
		return NewWPError("invalid_request", "Invalid personal data request.", map[string]any{"status": 500})
	}
	status := strings.TrimSpace(post.Status)
	if status != "request-pending" && status != "request-failed" {
		return NewWPError("expired_request", "This personal data request has expired.", map[string]any{"status": 500})
	}
	if key == "" {
		return NewWPError("missing_key", "The confirmation key is missing from this personal data request.", map[string]any{"status": 500})
	}
	expiration := UserRequestKeyExpiration
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("user_request_key_expiration", expiration); v != nil {
			switch t := v.(type) {
			case int:
				expiration = t
			case int64:
				expiration = int(t)
			case float64:
				expiration = int(t)
			case string:
				if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
					expiration = n
				}
			}
		}
	}
	mod := strings.TrimSpace(post.Modified)
	if mod == "" {
		mod = strings.TrimSpace(post.Date)
	}
	var keyTime time.Time
	if t, err := time.Parse("2006-01-02 15:04:05", mod); err == nil {
		keyTime = t.UTC()
	}
	if !VerifyFastHash(key, savedKey) {
		return NewWPError("invalid_key", "The confirmation key is invalid for this personal data request.", map[string]any{"status": 500})
	}
	if keyTime.IsZero() || time.Now().UTC().After(keyTime.Add(time.Duration(expiration)*time.Second)) {
		return NewWPError("expired_key", "The confirmation key has expired for this personal data request.", map[string]any{"status": 500})
	}
	return nil
}

// ConfirmUserRequest marks a pending request as request-confirmed (P6.46).
func ConfirmUserRequest(c *Context, requestID int64) error {
	if c == nil || c.DB == nil || requestID <= 0 {
		return fmt.Errorf("wp: ConfirmUserRequest: invalid state")
	}
	now := time.Now().UTC()
	stamp := now.Format("2006-01-02 15:04:05")
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET post_status = %s, post_modified = %s, post_modified_gmt = %s "+
			"WHERE ID = %d AND post_type = %s",
		"request-confirmed", stamp, stamp, requestID, "user_request",
	)
	if err != nil {
		return err
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		return err
	}
	if c.Meta != nil {
		_, _, _, _ = c.Meta.UpdateMetadata(MetaPost, requestID, "_wp_user_request_confirmed_timestamp", now.Unix(), "")
	}
	return nil
}

// PrivacyAccountRequestConfirmedMessage mirrors _wp_privacy_account_request_confirmed_message (P6.46).
func PrivacyAccountRequestConfirmedMessage(c *Context, requestID int64) string {
	message := `<p class="success">Action has been confirmed.</p>` +
		`<p>The site administrator has been notified and will fulfill your request as soon as possible.</p>`
	req, err := GetUserRequest(c, requestID)
	if err == nil && req != nil {
		switch req.ActionName {
		case "export_personal_data":
			message = `<p class="success">Thanks for confirming your export request.</p>` +
				`<p>The site administrator has been notified. You will receive a link to download your export via email when they fulfill your request.</p>`
		case "remove_personal_data":
			message = `<p class="success">Thanks for confirming your erasure request.</p>` +
				`<p>The site administrator has been notified. You will receive an email confirmation when they erase your data.</p>`
		}
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("user_request_action_confirmed_message", message, requestID); v != nil {
			if s, ok := v.(string); ok {
				message = s
			}
		}
	}
	return message
}
