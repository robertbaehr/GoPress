package rest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Taxonomy is a minimal wp/v2 taxonomy object (view context).
type Taxonomy struct {
	Name          string   `json:"name"`
	Slug          string   `json:"slug"`
	Description   string   `json:"description"`
	Types         []string `json:"types"`
	Hierarchical  bool     `json:"hierarchical"`
	RestBase      string   `json:"rest_base"`
	RestNamespace string   `json:"rest_namespace"`
}

// Term is a minimal wp/v2 category or tag object.
type Term struct {
	ID          int64  `json:"id"`
	Count       int64  `json:"count"`
	Description string `json:"description"`
	Link        string `json:"link"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Taxonomy    string `json:"taxonomy"`
	Parent      *int64 `json:"parent,omitempty"`
}

func (s *Server) serveTaxonomies(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	writeJSON(w, http.StatusOK, builtinTaxonomies())
}

func (s *Server) serveCategories(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.serveTermCollection(w, r, c, "category")
}

func (s *Server) serveTags(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.serveTermCollection(w, r, c, "post_tag")
}

func (s *Server) serveCategoryItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	s.serveTermItem(w, r, c, "category", id)
}

func (s *Server) serveTagItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	s.serveTermItem(w, r, c, "post_tag", id)
}

func (s *Server) serveTermCollection(w http.ResponseWriter, r *http.Request, c *wp.Context, taxonomy string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
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
	writeJSON(w, http.StatusOK, terms)
}

func (s *Server) serveTermItem(w http.ResponseWriter, r *http.Request, c *wp.Context, taxonomy string, id int64) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	term, err := getTermByID(c, taxonomy, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_term_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if term == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_term_invalid",
			"message": "Term does not exist.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	writeJSON(w, http.StatusOK, term)
}

func builtinTaxonomies() map[string]Taxonomy {
	return map[string]Taxonomy{
		"category": {
			Name:          "Categories",
			Slug:          "category",
			Description:   "",
			Types:         []string{"post"},
			Hierarchical:  true,
			RestBase:      "categories",
			RestNamespace: "wp/v2",
		},
		"post_tag": {
			Name:          "Tags",
			Slug:          "post_tag",
			Description:   "",
			Types:         []string{"post"},
			Hierarchical:  false,
			RestBase:      "tags",
			RestNamespace: "wp/v2",
		},
	}
}

func listTermsPaged(c *wp.Context, taxonomy string, limit, page int) ([]Term, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	q, args, err := c.DB.Prepare(
		"SELECT t.term_id, t.name, t.slug, tt.description, tt.parent, tt.count, tt.taxonomy FROM "+
			c.DB.Terms()+" t INNER JOIN "+c.DB.TermTaxonomy()+
			" tt ON t.term_id = tt.term_id WHERE tt.taxonomy = %s ORDER BY t.name ASC LIMIT %d OFFSET %d",
		taxonomy, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]Term, 0, len(rows))
	for _, row := range rows {
		out = append(out, termFromRow(row, home))
	}
	return out, nil
}

func countTerms(c *wp.Context, taxonomy string) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	q, args, err := c.DB.Prepare(
		"SELECT COUNT(*) AS total FROM "+c.DB.TermTaxonomy()+" WHERE taxonomy = %s",
		taxonomy,
	)
	if err != nil {
		return 0, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	return int(asInt64(row["total"])), nil
}

func getTermByID(c *wp.Context, taxonomy string, id int64) (*Term, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if id <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT t.term_id, t.name, t.slug, tt.description, tt.parent, tt.count, tt.taxonomy FROM "+
			c.DB.Terms()+" t INNER JOIN "+c.DB.TermTaxonomy()+
			" tt ON t.term_id = tt.term_id WHERE t.term_id = %d AND tt.taxonomy = %s LIMIT 1",
		id, taxonomy,
	)
	if err != nil {
		return nil, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	t := termFromRow(row, home)
	return &t, nil
}

func termFromRow(row map[string]any, home string) Term {
	id := asInt64(row["term_id"])
	slug := asString(row["slug"])
	taxonomy := asString(row["taxonomy"])
	parent := asInt64(row["parent"])
	t := Term{
		ID:          id,
		Count:       asInt64(row["count"]),
		Description: asString(row["description"]),
		Link:        termPermalink(home, taxonomy, slug),
		Name:        asString(row["name"]),
		Slug:        slug,
		Taxonomy:    taxonomy,
	}
	if taxonomy == "category" {
		p := parent
		t.Parent = &p
	}
	return t
}

func termPermalink(home, taxonomy, slug string) string {
	if slug == "" {
		return home + "/"
	}
	switch taxonomy {
	case "category":
		return fmt.Sprintf("%s/category/%s/", home, slug)
	case "post_tag":
		return fmt.Sprintf("%s/tag/%s/", home, slug)
	default:
		return fmt.Sprintf("%s/?taxonomy=%s&term=%s", home, taxonomy, slug)
	}
}
