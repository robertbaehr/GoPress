// Rewrite parsing (P3.2) — load rules and map request paths to query vars.
package wp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// RewriteRule is one WordPress rewrite pattern → query template pair.
type RewriteRule struct {
	Match string
	Query string
}

// RewriteRules is an ordered rule list (first match wins, like WP_Rewrite).
type RewriteRules []RewriteRule

// ParseResult is the outcome of matching a request path against rewrite rules.
type ParseResult struct {
	URL          string
	Request      string
	MatchedRule  string
	MatchedQuery string
	QueryVars    map[string]string
	IsHome       bool
}

// flagQueryVars are used to classify non-home requests (subset of WP public vars).
var flagQueryVars = []string{
	"p", "page_id", "name", "pagename", "category_name", "tag",
	"author_name", "feed", "year", "monthnum", "day", "attachment",
	"attachment_id", "rest_route", "error", "s", "embed",
}

// LoadRewriteRules returns rewrite rules from the rewrite_rules option, or
// generates a minimal set from permalink_structure when the option is empty.
func (c *Context) LoadRewriteRules() (RewriteRules, error) {
	if c == nil || c.Options == nil {
		return nil, fmt.Errorf("wp: LoadRewriteRules: nil Context")
	}
	raw := c.Options.GetOption("rewrite_rules", "")
	if s := optionValueString(raw); s != "" {
		rules, err := ParseRewriteRulesPHP(s)
		if err != nil {
			return nil, err
		}
		if len(rules) > 0 {
			return rules, nil
		}
	}
	structure := optionValueString(c.Options.GetOption("permalink_structure", ""))
	return GenerateRewriteRules(structure), nil
}

// ParseRewriteRulesPHP unserializes a PHP associative array of match → query,
// preserving key order (required for correct first-match behavior).
func ParseRewriteRulesPHP(serialized string) (RewriteRules, error) {
	pairs, err := phpUnserializeStringMapOrdered(serialized)
	if err != nil {
		return nil, fmt.Errorf("wp: rewrite_rules: %w", err)
	}
	out := make(RewriteRules, 0, len(pairs))
	for _, kv := range pairs {
		out = append(out, RewriteRule{Match: kv[0], Query: kv[1]})
	}
	return out, nil
}

// GenerateRewriteRules builds a minimal rule set for common permalink structures.
// Prefer DB-stored rewrite_rules when available; this is a fallback shim.
func GenerateRewriteRules(structure string) RewriteRules {
	structure = strings.TrimSpace(structure)
	var rules RewriteRules
	// REST + sitemap (always registered by modern WP).
	rules = append(rules,
		RewriteRule{Match: `^wp-json/?$`, Query: `index.php?rest_route=/`},
		RewriteRule{Match: `^wp-json/(.*)?`, Query: `index.php?rest_route=/$matches[1]`},
	)
	if structure == "" {
		return rules
	}
	rules = append(rules,
		RewriteRule{Match: `feed/(feed|rdf|rss|rss2|atom)/?$`, Query: `index.php?&feed=$matches[1]`},
		RewriteRule{Match: `(feed|rdf|rss|rss2|atom)/?$`, Query: `index.php?&feed=$matches[1]`},
		RewriteRule{Match: `comments/(feed|rdf|rss|rss2|atom)/?$`, Query: `index.php?&feed=$matches[1]&withcomments=1`},
		RewriteRule{Match: `page/?([0-9]{1,})/?$`, Query: `index.php?&paged=$matches[1]`},
		RewriteRule{Match: `category/(.+?)/?$`, Query: `index.php?category_name=$matches[1]`},
		RewriteRule{Match: `tag/([^/]+)/?$`, Query: `index.php?tag=$matches[1]`},
		RewriteRule{Match: `author/([^/]+)/?$`, Query: `index.php?author_name=$matches[1]`},
		// Attachment pretty permalinks (subset of WP attachment rewrite rules).
		RewriteRule{Match: `(.+)/attachment/([^/]+)/?$`, Query: `index.php?attachment=$matches[2]`},
		RewriteRule{Match: `attachment/([^/]+)/?$`, Query: `index.php?attachment=$matches[1]`},
	)
	switch {
	case strings.Contains(structure, "%post_id%"):
		rules = append(rules,
			RewriteRule{Match: `archives/([0-9]+)/?$`, Query: `index.php?p=$matches[1]`},
		)
	case strings.Contains(structure, "%year%") && strings.Contains(structure, "%day%"):
		rules = append(rules,
			RewriteRule{Match: `([0-9]{4})/([0-9]{1,2})/([0-9]{1,2})/([^/]+)(?:/([0-9]+))?/?$`, Query: `index.php?year=$matches[1]&monthnum=$matches[2]&day=$matches[3]&name=$matches[4]&page=$matches[5]`},
			RewriteRule{Match: `([0-9]{4})/([0-9]{1,2})/([0-9]{1,2})/?$`, Query: `index.php?year=$matches[1]&monthnum=$matches[2]&day=$matches[3]`},
			RewriteRule{Match: `([0-9]{4})/([0-9]{1,2})/?$`, Query: `index.php?year=$matches[1]&monthnum=$matches[2]`},
			RewriteRule{Match: `([0-9]{4})/?$`, Query: `index.php?year=$matches[1]`},
		)
	case strings.Contains(structure, "%year%") && strings.Contains(structure, "%monthnum%"):
		rules = append(rules,
			RewriteRule{Match: `([0-9]{4})/([0-9]{1,2})/([^/]+)(?:/([0-9]+))?/?$`, Query: `index.php?year=$matches[1]&monthnum=$matches[2]&name=$matches[3]&page=$matches[4]`},
			RewriteRule{Match: `([0-9]{4})/([0-9]{1,2})/?$`, Query: `index.php?year=$matches[1]&monthnum=$matches[2]`},
			RewriteRule{Match: `([0-9]{4})/?$`, Query: `index.php?year=$matches[1]`},
		)
	case strings.Contains(structure, "%category%") && strings.Contains(structure, "%postname%"):
		rules = append(rules,
			RewriteRule{Match: `blog/(.+?)/([^/]+)/?$`, Query: `index.php?category_name=$matches[1]&name=$matches[2]`},
		)
	case strings.Contains(structure, "%postname%"):
		rules = append(rules,
			RewriteRule{Match: `([^/]+)(?:/([0-9]+))?/?$`, Query: `index.php?name=$matches[1]&page=$matches[2]`},
		)
	}
	// Catch-all page path last (WP registers pagename rules late for pretty permalinks).
	rules = append(rules,
		RewriteRule{Match: `(.?.+?)(?:/([0-9]+))?/?$`, Query: `index.php?pagename=$matches[1]&page=$matches[2]`},
	)
	return rules
}

// ParseRequestPath matches path (+ optional raw query) against rules.
// pathURL may be a path ("/foo/") or path with query ("/?p=1").
func (rules RewriteRules) ParseRequestPath(pathURL string) *ParseResult {
	return rules.ParseRequestPathWith(pathURL, nil)
}

// PageExistsFunc reports whether a published page exists for the given pagename path
// (WordPress verbose page rules). When nil, pagename matches are accepted unconditionally.
type PageExistsFunc func(pagename string) bool

// ParseRequestPathWith is ParseRequestPath plus optional verbose page-rule checks.
func (rules RewriteRules) ParseRequestPathWith(pathURL string, pageExists PageExistsFunc) *ParseResult {
	raw := pathURL
	path := pathURL
	query := ""
	if i := strings.Index(path, "?"); i >= 0 {
		query = path[i+1:]
		path = path[:i]
	}
	if strings.HasPrefix(strings.ToLower(path), "http://") || strings.HasPrefix(strings.ToLower(path), "https://") {
		if u, err := url.Parse(path); err == nil {
			path = u.Path
			if query == "" {
				query = u.RawQuery
			}
		}
	}
	if path == "" {
		path = "/"
	}
	request := strings.Trim(path, "/")

	res := &ParseResult{
		URL:       raw,
		Request:   request,
		QueryVars: map[string]string{},
	}

	for _, rule := range rules {
		re, err := regexp.Compile("^" + rule.Match)
		if err != nil {
			continue
		}
		m := re.FindStringSubmatch(request)
		if m == nil {
			// WP also tries urldecoded request.
			if dec, err := url.PathUnescape(request); err == nil && dec != request {
				m = re.FindStringSubmatch(dec)
			}
		}
		if m == nil {
			continue
		}
		filled := replaceMatchRefs(rule.Query, m)
		vars := parseIndexPHPQuery(filled)
		// Verbose page rules: skip pagename matches when the page does not exist
		// so later taxonomy/date/post rules can win (WP_Rewrite::$use_verbose_page_rules).
		if pageExists != nil && strings.Contains(rule.Query, "pagename=") {
			if pn := vars["pagename"]; pn != "" && !pageExists(pn) {
				continue
			}
		}
		res.MatchedRule = rule.Match
		res.MatchedQuery = rule.Query
		res.QueryVars = vars
		break
	}

	if query != "" {
		if q, err := url.ParseQuery(query); err == nil {
			for k, vs := range q {
				if len(vs) > 0 {
					res.QueryVars[k] = vs[0]
				}
			}
		}
	}

	res.IsHome = request == "" && !hasContentQueryVar(res.QueryVars)
	return res
}

// ParseRequest parses r.URL against rules and updates Scope conditionals.
func (c *Context) ParseRequest(rules RewriteRules) *ParseResult {
	path := "/"
	rawQuery := ""
	if c != nil && c.Req != nil && c.Req.URL != nil {
		path = c.Req.URL.Path
		rawQuery = c.Req.URL.RawQuery
	}
	u := path
	if rawQuery != "" {
		u = path + "?" + rawQuery
	}
	var pageExists PageExistsFunc
	if c != nil {
		pageExists = c.pageExists
	}
	res := rules.ParseRequestPathWith(u, pageExists)
	if c != nil && c.Scope != nil {
		c.Scope.mu.Lock()
		c.Scope.IsHome = res.IsHome
		c.Scope.IsFrontPage = res.IsHome
		c.Scope.IsSingular = hasAny(res.QueryVars, "p", "page_id", "name", "pagename", "attachment", "attachment_id")
		c.Scope.IsSingle = hasAny(res.QueryVars, "p", "name", "attachment", "attachment_id")
		c.Scope.IsPage = hasAny(res.QueryVars, "page_id", "pagename")
		c.Scope.IsArchive = hasAny(res.QueryVars, "year", "monthnum", "day", "category_name", "tag", "author_name")
		c.Scope.IsFeed = hasAny(res.QueryVars, "feed")
		c.Scope.IsSearch = hasAny(res.QueryVars, "s")
		c.Scope.SearchQuery = res.QueryVars["s"]
		c.Scope.IsREST = hasAny(res.QueryVars, "rest_route")
		// Date archives (P3.22): year/month/day without a singular post name.
		if !c.Scope.IsSingular && hasAny(res.QueryVars, "year", "monthnum", "day") {
			c.Scope.IsDate = true
			c.Scope.DateYear = atoiScope(res.QueryVars["year"])
			c.Scope.DateMonth = atoiScope(res.QueryVars["monthnum"])
			c.Scope.DateDay = atoiScope(res.QueryVars["day"])
			c.Scope.IsYear = c.Scope.DateYear > 0 && c.Scope.DateMonth == 0 && c.Scope.DateDay == 0
			c.Scope.IsMonth = c.Scope.DateYear > 0 && c.Scope.DateMonth > 0 && c.Scope.DateDay == 0
			c.Scope.IsDay = c.Scope.DateYear > 0 && c.Scope.DateMonth > 0 && c.Scope.DateDay > 0
		}
		// Author archives (P3.23).
		if !c.Scope.IsSingular && hasAny(res.QueryVars, "author_name", "author") {
			c.Scope.IsAuthor = true
			c.Scope.AuthorName = res.QueryVars["author_name"]
			if id, err := strconv.ParseInt(res.QueryVars["author"], 10, 64); err == nil && id > 0 {
				c.Scope.AuthorID = id
			}
		}
		if p := res.QueryVars["p"]; p != "" {
			if id, err := strconv.ParseInt(p, 10, 64); err == nil {
				c.Scope.QueriedObjectID = id
			}
		} else if p := res.QueryVars["page_id"]; p != "" {
			if id, err := strconv.ParseInt(p, 10, 64); err == nil {
				c.Scope.QueriedObjectID = id
			}
		} else if p := res.QueryVars["attachment_id"]; p != "" {
			if id, err := strconv.ParseInt(p, 10, 64); err == nil {
				c.Scope.QueriedObjectID = id
			}
		}
		c.Scope.mu.Unlock()
	}
	return res
}

func replaceMatchRefs(query string, matches []string) string {
	return matchRefPattern.ReplaceAllStringFunc(query, func(s string) string {
		m := matchRefPattern.FindStringSubmatch(s)
		if len(m) < 2 {
			return ""
		}
		i, err := strconv.Atoi(m[1])
		if err != nil || i < 0 || i >= len(matches) {
			return ""
		}
		return matches[i]
	})
}

var matchRefPattern = regexp.MustCompile(`\$matches\[(\d+)\]`)

// parseIndexPHPQuery turns "index.php?foo=bar&baz=$x" into query vars.
// Unlike PHP parse_str on the raw string, the index.php? prefix is stripped.
func parseIndexPHPQuery(q string) map[string]string {
	out := map[string]string{}
	q = strings.TrimSpace(q)
	if q == "" {
		return out
	}
	if i := strings.Index(q, "?"); i >= 0 {
		q = q[i+1:]
	} else if strings.HasPrefix(q, "index.php") {
		return out
	}
	vals, err := url.ParseQuery(q)
	if err != nil {
		return out
	}
	for k, vs := range vals {
		if len(vs) > 0 {
			out[k] = vs[0]
		}
	}
	return out
}

func hasContentQueryVar(vars map[string]string) bool {
	for _, k := range flagQueryVars {
		if v, ok := vars[k]; ok && v != "" {
			return true
		}
	}
	return false
}

func hasAny(vars map[string]string, keys ...string) bool {
	for _, k := range keys {
		if v, ok := vars[k]; ok && v != "" {
			return true
		}
	}
	return false
}

func atoiScope(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// pageExists reports whether a published page exists at pagename (hierarchical path).
func (c *Context) pageExists(pagename string) bool {
	if c == nil || c.DB == nil || pagename == "" {
		return false
	}
	// Match final path segment as post_name; hierarchical resolution lands in a later task.
	parts := strings.Split(pagename, "/")
	slug := parts[len(parts)-1]
	q, args, err := c.DB.Prepare(
		"SELECT ID FROM "+c.DB.Posts()+" WHERE post_type = %s AND post_status = %s AND post_name = %s LIMIT 1",
		"page", "publish", slug,
	)
	if err != nil {
		return false
	}
	row, err := c.DB.GetRow(q, args...)
	return err == nil && row != nil
}

// phpUnserializeStringMapOrdered parses a:N:{s:…;s:…;…} preserving order.
func phpUnserializeStringMapOrdered(data string) ([][2]string, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, nil
	}
	if !strings.HasPrefix(data, "a:") {
		return nil, fmt.Errorf("php unserialize ordered: not an array")
	}
	// a:N:{
	rest := data[2:]
	colon := strings.IndexByte(rest, ':')
	if colon < 0 {
		return nil, fmt.Errorf("php unserialize ordered: bad length")
	}
	n, err := strconv.Atoi(rest[:colon])
	if err != nil {
		return nil, err
	}
	rest = rest[colon+1:]
	if !strings.HasPrefix(rest, "{") {
		return nil, fmt.Errorf("php unserialize ordered: expected '{'")
	}
	pos := 1 // inside {
	out := make([][2]string, 0, n)
	for i := 0; i < n; i++ {
		key, np, err := phpParseStringAt(rest, pos)
		if err != nil {
			return nil, err
		}
		pos = np
		val, np, err := phpParseStringAt(rest, pos)
		if err != nil {
			return nil, err
		}
		pos = np
		out = append(out, [2]string{key, val})
	}
	return out, nil
}

func phpParseStringAt(data string, i int) (string, int, error) {
	if i >= len(data) || data[i] != 's' {
		return "", i, fmt.Errorf("php unserialize ordered: expected string at %d", i)
	}
	v, ni, err := phpParse(data, i)
	if err != nil {
		return "", i, err
	}
	s, ok := v.(string)
	if !ok {
		return "", i, fmt.Errorf("php unserialize ordered: not a string")
	}
	return s, ni, nil
}

// LoadRewriteRulesJSONFile loads ordered rules from a golden rules-*.json file.
// Used by tests; JSON object key order is preserved via Decoder.
func LoadRewriteRulesJSONFile(path string) (RewriteRules, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	tok, err := dec.Token()
	if err != nil {
		return nil, "", err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, "", fmt.Errorf("expected object")
	}
	var structure string
	var rules RewriteRules
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, "", err
		}
		key, _ := keyTok.(string)
		switch key {
		case "structure":
			if err := dec.Decode(&structure); err != nil {
				return nil, "", err
			}
		case "rules":
			t, err := dec.Token()
			if err != nil {
				return nil, "", err
			}
			if d, ok := t.(json.Delim); !ok || d != '{' {
				// empty array []
				if d, ok := t.(json.Delim); ok && d == '[' {
					for dec.More() {
						var skip any
						_ = dec.Decode(&skip)
					}
					_, _ = dec.Token() // ]
					continue
				}
				return nil, "", fmt.Errorf("rules: expected object")
			}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, "", err
				}
				k, _ := kt.(string)
				var v string
				if err := dec.Decode(&v); err != nil {
					return nil, "", err
				}
				rules = append(rules, RewriteRule{Match: k, Query: v})
			}
			if _, err := dec.Token(); err != nil { // }
				return nil, "", err
			}
		default:
			var skip any
			if err := dec.Decode(&skip); err != nil {
				return nil, "", err
			}
		}
	}
	return rules, structure, nil
}

// GoldenRewriteDir returns tests/golden/rewrite relative to a module root marker.
func GoldenRewriteDir() string {
	// Walk up from cwd / this file's compile-time path is unavailable; use cwd walk.
	dir, err := os.Getwd()
	if err != nil {
		return "tests/golden/rewrite"
	}
	for {
		candidate := filepath.Join(dir, "tests", "golden", "rewrite")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "tests/golden/rewrite"
}
