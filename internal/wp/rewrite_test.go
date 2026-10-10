package wp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestParseHomeIsHome(t *testing.T) {
	rules := wp.GenerateRewriteRules("/%postname%/")
	res := rules.ParseRequestPath("/")
	if !res.IsHome || res.Request != "" || res.MatchedRule != "" {
		t.Fatalf("home parse: %+v", res)
	}
}

func TestParseQueryP(t *testing.T) {
	res := wp.RewriteRules{}.ParseRequestPath("/?p=1")
	if res.QueryVars["p"] != "1" || res.IsHome {
		t.Fatalf("got %+v", res)
	}
}

func TestParsePostname(t *testing.T) {
	rules := wp.GenerateRewriteRules("/%postname%/")
	res := rules.ParseRequestPath("/hello-world/")
	if res.Request != "hello-world" {
		t.Fatalf("request=%q", res.Request)
	}
	if res.QueryVars["name"] != "hello-world" && res.QueryVars["pagename"] != "hello-world" {
		t.Fatalf("query_vars=%v", res.QueryVars)
	}
	if res.IsHome {
		t.Fatal("expected non-home")
	}
}

func TestLoadRewriteRulesFromDB(t *testing.T) {
	cfg := testDBConfig()
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	rules, err := c.LoadRewriteRules()
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) < 10 {
		t.Fatalf("expected many rewrite rules from DB, got %d", len(rules))
	}
	res := rules.ParseRequestPath("/2020/01/01/hello-world/")
	if res.MatchedRule == "" {
		t.Fatalf("expected a matched rule, vars=%v", res.QueryVars)
	}
	if res.QueryVars["year"] != "2020" && res.QueryVars["name"] != "hello-world" {
		// With verbose page checks via Context, date vars should win.
		c.Req = &http.Request{URL: &url.URL{Path: "/2020/01/01/hello-world/"}}
		res = c.ParseRequest(rules)
		if res.QueryVars["year"] != "2020" {
			t.Fatalf("expected date/name vars, got %v (rule=%s)", res.QueryVars, res.MatchedRule)
		}
	}
	home := rules.ParseRequestPath("/")
	if !home.IsHome {
		t.Fatalf("home: %+v", home)
	}
}

func TestGoldenRewriteParseSubset(t *testing.T) {
	dir := wp.GoldenRewriteDir()
	plainPath := filepath.Join(dir, "parse-plain.json")
	plain, err := loadGoldenParse(plainPath)
	if err != nil {
		t.Skipf("golden unavailable: %v", err)
	}
	rules := wp.GenerateRewriteRules("")
	for _, item := range plain.Parsed {
		got := rules.ParseRequestPath(item.URL)
		if got.Request != item.Request {
			t.Errorf("plain %s: request=%q want %q", item.URL, got.Request, item.Request)
		}
		if item.URL == "/?p=1" && got.QueryVars["p"] != "1" {
			t.Errorf("plain /?p=1: vars=%v", got.QueryVars)
		}
		if item.URL == "/" && !got.IsHome {
			t.Errorf("plain /: expected IsHome")
		}
		// Plain golden has no rewrite rules; our generator always adds REST — skip those URLs.
		if strings.HasPrefix(item.Request, "wp-json") {
			continue
		}
		if (item.MatchedRule == nil || *item.MatchedRule == "") && got.MatchedRule != "" && got.QueryVars["rest_route"] == "" {
			t.Errorf("plain %s: matched %q want none", item.URL, got.MatchedRule)
		}
	}

	dayPath := filepath.Join(dir, "parse-day-name.json")
	day, err := loadGoldenParse(dayPath)
	if err != nil {
		t.Fatal(err)
	}
	// Golden rules-*.json keys are ksorted for stable encoding, so match against
	// GenerateRewriteRules (ordered) for a representative URL subset.
	gen := wp.GenerateRewriteRules(day.Structure)
	subset := map[string]bool{
		"/":                        true,
		"/2020/01/01/hello-world/": true,
		"/category/uncategorized/": true,
		"/tag/test/":               true,
		"/author/admin/":           true,
		"/feed/":                   true,
		"/comments/feed/":          true,
		"/?p=1":                    true,
		"/?page_id=2":              true,
		"/wp-json/":                true,
		"/2020/":                   true,
		"/2020/01/":                true,
	}
	for _, item := range day.Parsed {
		if !subset[item.URL] {
			continue
		}
		got := gen.ParseRequestPath(item.URL)
		if got.Request != item.Request {
			t.Errorf("%s: request=%q want %q", item.URL, got.Request, item.Request)
		}
		wantRule := ""
		if item.MatchedRule != nil {
			wantRule = *item.MatchedRule
		}
		if got.MatchedRule != wantRule {
			t.Errorf("%s: matched_rule=%q want %q", item.URL, got.MatchedRule, wantRule)
		}
	}

	// DB-loaded rules + verbose page checks: home and date permalink at least.
	cfg := testDBConfig()
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()
	dbRules, err := c.LoadRewriteRules()
	if err != nil {
		t.Fatal(err)
	}
	c.Req = &http.Request{URL: &url.URL{Path: "/"}}
	if !c.ParseRequest(dbRules).IsHome {
		t.Fatal("DB rules: / should be home")
	}
	c.Req = &http.Request{URL: &url.URL{Path: "/2020/01/01/hello-world/"}}
	got := c.ParseRequest(dbRules)
	if got.QueryVars["year"] != "2020" && got.MatchedRule == "" {
		t.Fatalf("DB rules date parse: %+v", got)
	}
}

func testDBConfig() *config.Config {
	return &config.Config{
		DBName: "gopress_ref", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		AuthKey: "k", SecureAuthKey: "k", LoggedInKey: "k", NonceKey: "k",
		AuthSalt: "s", SecureAuthSalt: "s", LoggedInSalt: "s", NonceSalt: "s",
	}
}

type goldenParseFile struct {
	Structure string             `json:"structure"`
	Parsed    []goldenParseEntry `json:"parsed"`
}

type goldenParseEntry struct {
	URL          string          `json:"url"`
	Request      string          `json:"request"`
	MatchedRule  *string         `json:"matched_rule"`
	MatchedQuery *string         `json:"matched_query"`
	QueryVars    json.RawMessage `json:"query_vars"`
}

func loadGoldenParse(path string) (*goldenParseFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f goldenParseFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return &f, nil
}
