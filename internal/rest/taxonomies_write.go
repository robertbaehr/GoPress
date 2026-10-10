package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

type termWriteBody struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Parent      *int64 `json:"parent"`
}

func (s *Server) serveTermCollection(w http.ResponseWriter, r *http.Request, c *wp.Context, taxonomy string) {
	switch r.Method {
	case http.MethodGet:
		s.listTermCollection(w, r, c, taxonomy)
	case http.MethodPost:
		s.createTerm(w, r, c, taxonomy)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func (s *Server) listTermCollection(w http.ResponseWriter, r *http.Request, c *wp.Context, taxonomy string) {
	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	total, err := countTerms(c, taxonomy)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_term_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	terms, err := listTermsPaged(c, taxonomy, perPage, page)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_term_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	setCollectionHeaders(w, r, c, total, totalPages, page)
	if canManageTerms(c) {
		w.Header().Set("Allow", "GET, POST")
	}
	writeJSON(w, http.StatusOK, terms)
}

// createTerm handles POST /wp/v2/categories and /wp/v2/tags (P4.44).
func (s *Server) createTerm(w http.ResponseWriter, r *http.Request, c *wp.Context, taxonomy string) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to create terms in this taxonomy.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !canManageTerms(c) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to create terms in this taxonomy.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	body, err := decodeTermWriteBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_missing_name",
			"message": "Sorry, you must include a name for the term.",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	slug := strings.TrimSpace(body.Slug)
	if slug == "" {
		slug = wp.SanitizeTitle(name)
	} else {
		slug = wp.SanitizeTitle(slug)
	}
	if slug == "" {
		slug = "term"
	}
	slug, err = uniqueTermSlug(c, slug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_term_create",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	var parent int64
	if taxonomy == "category" && body.Parent != nil && *body.Parent > 0 {
		parent = *body.Parent
	}
	id, err := insertTermRow(c, taxonomy, name, slug, body.Description, parent)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_term_create",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	term, err := getTermByID(c, taxonomy, id)
	if err != nil || term == nil {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusCreated, term)
}

func canManageTerms(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	return c.UserCan(c.CurrentUser, "manage_categories")
}

func decodeTermWriteBody(r *http.Request) (termWriteBody, error) {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return termWriteBody{}, err
	}
	var body termWriteBody
	if len(raw) == 0 {
		return body, nil
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return termWriteBody{}, err
	}
	return body, nil
}

func uniqueTermSlug(c *wp.Context, base string) (string, error) {
	slug := base
	for i := 2; i < 1000; i++ {
		exists, err := termSlugExists(c, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	return "", fmt.Errorf("could not allocate unique slug")
}

func termSlugExists(c *wp.Context, slug string) (bool, error) {
	if c == nil || c.DB == nil {
		return false, fmt.Errorf("nil DB")
	}
	q, args, err := c.DB.Prepare(
		"SELECT term_id FROM "+c.DB.Terms()+" WHERE slug = %s LIMIT 1",
		slug,
	)
	if err != nil {
		return false, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return false, err
	}
	return row != nil, nil
}

func insertTermRow(c *wp.Context, taxonomy, name, slug, description string, parent int64) (int64, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Terms()+" (name, slug, term_group) VALUES (%s, %s, %d)",
		name, slug, 0,
	)
	if err != nil {
		return 0, err
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	termID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	tq, targs, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.TermTaxonomy()+
			" (term_id, taxonomy, description, parent, count) VALUES (%d, %s, %s, %d, %d)",
		termID, taxonomy, description, parent, 0,
	)
	if err != nil {
		return 0, err
	}
	if _, err := c.DB.Exec(tq, targs...); err != nil {
		return 0, err
	}
	return termID, nil
}
