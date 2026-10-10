package wp

import (
	"fmt"
	"strings"
)

// PersonalDataEraseResult summarizes a privacy erasure stub run (P8.5).
type PersonalDataEraseResult struct {
	Email              string `json:"email"`
	CommentsAnonymized int    `json:"comments_anonymized"`
	UserAnonymized     bool   `json:"user_anonymized"`
	Message            string `json:"message"`
}

// ErasePersonalData anonymizes comments (and optional user profile fields) for email (P8.5).
// Full request workflow / eraser plugins land later; this primes the erasure shape.
func ErasePersonalData(c *Context, email string) (*PersonalDataEraseResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("wp: invalid erase email")
	}
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: ErasePersonalData: invalid state")
	}

	res := &PersonalDataEraseResult{
		Email:   email,
		Message: "GoPress privacy erasure stub — full eraser groups land later.",
	}

	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Comments()+" SET comment_author = %s, comment_author_email = %s, "+
			"comment_author_url = %s, comment_author_IP = %s, comment_agent = %s "+
			"WHERE comment_author_email = %s",
		"Anonymous",
		"",
		"",
		"",
		"",
		email,
	)
	if err != nil {
		return nil, err
	}
	execRes, err := c.DB.Exec(q, args...)
	if err != nil {
		return nil, err
	}
	if n, err := execRes.RowsAffected(); err == nil {
		res.CommentsAnonymized = int(n)
	}

	// Anonymize a matching non-admin user profile (login retained for FK integrity).
	u, err := c.GetUserBy("email", email)
	if err == nil && u != nil && u.Login != "admin" {
		uq, uargs, uerr := c.DB.Prepare(
			"UPDATE "+c.DB.Users()+" SET user_email = %s, user_url = %s, display_name = %s WHERE ID = %d",
			"deleted@site.invalid",
			"",
			"Anonymous",
			u.ID,
		)
		if uerr == nil {
			if _, uerr = c.DB.Exec(uq, uargs...); uerr == nil {
				res.UserAnonymized = true
			}
		}
	}

	res.Message = fmt.Sprintf(
		"Erased personal data for %s (%d comments anonymized).",
		email, res.CommentsAnonymized,
	)
	return res, nil
}
