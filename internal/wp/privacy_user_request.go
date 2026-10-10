package wp

import (
	"fmt"
	"strings"
	"time"
)

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
