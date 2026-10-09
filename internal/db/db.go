// Package db is the WordPress $wpdb port foundation (P1.3).
//
// SQL must use placeholders (? after Prepare, or Prepare's %s/%d/%f).
// Never concatenate untrusted values into query strings.
package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/robertbaehr/GoPress/internal/config"

	_ "github.com/go-sql-driver/mysql" // MySQL/MariaDB driver
)

// DB wraps *sql.DB with WordPress table-prefix helpers.
type DB struct {
	SQL         *sql.DB
	TablePrefix string
	Charset     string
}

// Open connects using cfg.DSN() and pings the server.
func Open(cfg *config.Config) (*DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("db: nil config")
	}
	return OpenDSN(cfg.DSN(), cfg.TablePrefix, cfg.DBCharset)
}

// OpenDSN connects with an explicit DSN (e.g. GOPRESS_TEST_DSN).
func OpenDSN(dsn, tablePrefix, charset string) (*DB, error) {
	if tablePrefix == "" {
		tablePrefix = "wp_"
	}
	if charset == "" {
		charset = "utf8mb4"
	}
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return &DB{SQL: sqlDB, TablePrefix: tablePrefix, Charset: charset}, nil
}

// Close closes the underlying *sql.DB.
func (d *DB) Close() error {
	if d == nil || d.SQL == nil {
		return nil
	}
	return d.SQL.Close()
}

// Table returns prefix + name (e.g. "wp_" + "posts").
func (d *DB) Table(name string) string {
	prefix := "wp_"
	if d != nil && d.TablePrefix != "" {
		prefix = d.TablePrefix
	}
	return prefix + name
}

func (d *DB) Posts() string             { return d.Table("posts") }
func (d *DB) Options() string           { return d.Table("options") }
func (d *DB) Comments() string          { return d.Table("comments") }
func (d *DB) CommentMeta() string       { return d.Table("commentmeta") }
func (d *DB) Links() string             { return d.Table("links") }
func (d *DB) PostMeta() string          { return d.Table("postmeta") }
func (d *DB) Terms() string             { return d.Table("terms") }
func (d *DB) TermTaxonomy() string      { return d.Table("term_taxonomy") }
func (d *DB) TermRelationships() string { return d.Table("term_relationships") }
func (d *DB) TermMeta() string          { return d.Table("termmeta") }
func (d *DB) Users() string             { return d.Table("users") }
func (d *DB) UserMeta() string          { return d.Table("usermeta") }

// EscapeLike escapes LIKE wildcards (_, %, \) for use inside a %s placeholder.
// Mirrors wpdb::esc_like.
func EscapeLike(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		switch r {
		case '\\', '%', '_':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// EscapeLike is the method form of the package helper.
func (d *DB) EscapeLike(text string) string { return EscapeLike(text) }

// Prepare converts WordPress-style %s / %d / %f placeholders to database/sql "?".
// Arguments are never concatenated into the SQL string.
// Literal percent signs are written as %% in the input query.
func Prepare(query string, args ...any) (string, []any, error) {
	// Match wpdb::prepare quote stripping for plain %s.
	query = strings.ReplaceAll(query, "'%s'", "%s")
	query = strings.ReplaceAll(query, `"%s"`, "%s")

	var b strings.Builder
	b.Grow(len(query))
	out := make([]any, 0, len(args))
	argIdx := 0

	for i := 0; i < len(query); i++ {
		if query[i] != '%' {
			b.WriteByte(query[i])
			continue
		}
		if i+1 >= len(query) {
			return "", nil, fmt.Errorf("db: Prepare: trailing %% in query")
		}
		switch query[i+1] {
		case '%':
			b.WriteByte('%')
			i++
		case 's', 'd', 'f':
			if argIdx >= len(args) {
				return "", nil, fmt.Errorf("db: Prepare: not enough arguments for placeholders")
			}
			b.WriteByte('?')
			out = append(out, args[argIdx])
			argIdx++
			i++
		default:
			return "", nil, fmt.Errorf("db: Prepare: unsupported placeholder %%%c (use %%s, %%d, %%f, or %%%%)", query[i+1])
		}
	}
	if argIdx != len(args) {
		return "", nil, fmt.Errorf("db: Prepare: %d unused argument(s)", len(args)-argIdx)
	}
	return b.String(), out, nil
}

// Prepare is the method form of the package helper.
func (d *DB) Prepare(query string, args ...any) (string, []any, error) {
	return Prepare(query, args...)
}

// Query runs a parameterized query (? placeholders).
func (d *DB) Query(query string, args ...any) (*sql.Rows, error) {
	if d == nil || d.SQL == nil {
		return nil, fmt.Errorf("db: nil DB")
	}
	return d.SQL.Query(query, args...)
}

// Exec runs a parameterized statement (? placeholders).
func (d *DB) Exec(query string, args ...any) (sql.Result, error) {
	if d == nil || d.SQL == nil {
		return nil, fmt.Errorf("db: nil DB")
	}
	return d.SQL.Exec(query, args...)
}

// GetVar returns the first column of the first row, or nil if empty.
func (d *DB) GetVar(query string, args ...any) (any, error) {
	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var v any
	if err := rows.Scan(&v); err != nil {
		return nil, err
	}
	if b, ok := v.([]byte); ok {
		return string(b), nil
	}
	return v, nil
}

// GetRow returns the first row as a column-name map, or nil if empty.
func (d *DB) GetRow(query string, args ...any) (map[string]any, error) {
	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		return nil, rows.Err()
	}
	return scanMap(rows, cols)
}

// GetResults returns all rows as column-name maps.
func (d *DB) GetResults(query string, args ...any) ([]map[string]any, error) {
	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		m, err := scanMap(rows, cols)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func scanMap(rows *sql.Rows, cols []string) (map[string]any, error) {
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	m := make(map[string]any, len(cols))
	for i, c := range cols {
		v := vals[i]
		if b, ok := v.([]byte); ok {
			m[c] = string(b)
		} else {
			m[c] = v
		}
	}
	return m, nil
}
