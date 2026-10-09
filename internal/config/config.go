// Package config loads WordPress-compatible site configuration (P1.2).
//
// Sources (later overrides earlier):
//  1. defaults
//  2. wp-config.php define() / $table_prefix assignments (subset parser)
//  3. environment variables (GOPRESS_* and selected WP_* names)
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Config is the loaded site configuration.
type Config struct {
	ABSPATH string

	DBName     string
	DBUser     string
	DBPassword string
	DBHost     string
	DBCharset  string
	DBCollate  string

	TablePrefix string

	AuthKey        string
	SecureAuthKey  string
	LoggedInKey    string
	NonceKey       string
	AuthSalt       string
	SecureAuthSalt string
	LoggedInSalt   string
	NonceSalt      string

	EnvironmentType string
	Debug           bool
	DisableWPCron   bool
	BlockExternal   bool
	AutoUpdaterOff  bool

	// Extra holds other define() constants as strings/bools for later tasks.
	Extra map[string]string
}

// LoadOptions controls Load.
type LoadOptions struct {
	// Root is ABSPATH (directory containing wp-config.php).
	Root string
	// ConfigPath overrides Root/wp-config.php when non-empty.
	ConfigPath string
	// SkipEnv disables environment overrides (tests).
	SkipEnv bool
}

// Load reads configuration from wp-config.php and environment.
func Load(opts LoadOptions) (*Config, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	cfg := defaults(root)

	path := opts.ConfigPath
	if path == "" {
		path = filepath.Join(root, "wp-config.php")
	}
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := applyPHPConfig(cfg, string(body)); err != nil {
			return nil, err
		}
	}

	if !opts.SkipEnv {
		applyEnv(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaults(root string) *Config {
	return &Config{
		ABSPATH:         ensureSlash(root),
		DBHost:          "localhost",
		DBCharset:       "utf8mb4",
		DBCollate:       "",
		TablePrefix:     "wp_",
		EnvironmentType: "production",
		Extra:           map[string]string{},
	}
}

// Validate ensures required DB and salt fields are present.
func (c *Config) Validate() error {
	var missing []string
	req := []struct {
		name, val string
	}{
		{"DB_NAME", c.DBName},
		{"DB_USER", c.DBUser},
		{"AUTH_KEY", c.AuthKey},
		{"SECURE_AUTH_KEY", c.SecureAuthKey},
		{"LOGGED_IN_KEY", c.LoggedInKey},
		{"NONCE_KEY", c.NonceKey},
		{"AUTH_SALT", c.AuthSalt},
		{"SECURE_AUTH_SALT", c.SecureAuthSalt},
		{"LOGGED_IN_SALT", c.LoggedInSalt},
		{"NONCE_SALT", c.NonceSalt},
	}
	for _, r := range req {
		if strings.TrimSpace(r.val) == "" {
			missing = append(missing, r.name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("config: missing required values: %s", strings.Join(missing, ", "))
	}
	if c.TablePrefix == "" {
		return fmt.Errorf("config: empty table_prefix")
	}
	return nil
}

// DSN returns a go-sql-driver/mysql DSN (P1.3).
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=false&charset=%s&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBName, c.DBCharset)
}

// Salt returns the salt string for a scheme name (auth, secure_auth, logged_in, nonce).
func (c *Config) Salt(scheme string) string {
	switch scheme {
	case "auth":
		return c.AuthKey + c.AuthSalt
	case "secure_auth":
		return c.SecureAuthKey + c.SecureAuthSalt
	case "logged_in":
		return c.LoggedInKey + c.LoggedInSalt
	case "nonce":
		return c.NonceKey + c.NonceSalt
	default:
		return c.AuthKey + c.AuthSalt
	}
}

var (
	reDefine = regexp.MustCompile(`(?i)define\s*\(\s*['"]([A-Z0-9_]+)['"]\s*,\s*(.+?)\s*\)\s*;`)
	rePrefix = regexp.MustCompile(`(?i)\$table_prefix\s*=\s*['"]([^'"]*)['"]\s*;`)
)

func applyPHPConfig(cfg *Config, src string) error {
	src = stripPHPComments(src)
	if m := rePrefix.FindStringSubmatch(src); len(m) == 2 {
		cfg.TablePrefix = m[1]
	}
	for _, m := range reDefine.FindAllStringSubmatch(src, -1) {
		name := m[1]
		raw := strings.TrimSpace(m[2])
		val, err := phpScalar(raw)
		if err != nil {
			// Keep going; unknown expressions stored raw in Extra.
			cfg.Extra[name] = raw
			continue
		}
		if err := setKnown(cfg, name, val); err != nil {
			return err
		}
		cfg.Extra[name] = val
	}
	return nil
}

func setKnown(cfg *Config, name, val string) error {
	switch name {
	case "DB_NAME":
		cfg.DBName = val
	case "DB_USER":
		cfg.DBUser = val
	case "DB_PASSWORD":
		cfg.DBPassword = val
	case "DB_HOST":
		cfg.DBHost = val
	case "DB_CHARSET":
		cfg.DBCharset = val
	case "DB_COLLATE":
		cfg.DBCollate = val
	case "AUTH_KEY":
		cfg.AuthKey = val
	case "SECURE_AUTH_KEY":
		cfg.SecureAuthKey = val
	case "LOGGED_IN_KEY":
		cfg.LoggedInKey = val
	case "NONCE_KEY":
		cfg.NonceKey = val
	case "AUTH_SALT":
		cfg.AuthSalt = val
	case "SECURE_AUTH_SALT":
		cfg.SecureAuthSalt = val
	case "LOGGED_IN_SALT":
		cfg.LoggedInSalt = val
	case "NONCE_SALT":
		cfg.NonceSalt = val
	case "WP_ENVIRONMENT_TYPE":
		cfg.EnvironmentType = val
	case "WP_DEBUG":
		cfg.Debug = phpBool(val)
	case "DISABLE_WP_CRON":
		cfg.DisableWPCron = phpBool(val)
	case "WP_HTTP_BLOCK_EXTERNAL":
		cfg.BlockExternal = phpBool(val)
	case "AUTOMATIC_UPDATER_DISABLED":
		cfg.AutoUpdaterOff = phpBool(val)
	case "ABSPATH":
		// Prefer explicit root from LoadOptions; ignore __DIR__ expressions.
		if !strings.Contains(val, "__DIR__") && val != "" {
			cfg.ABSPATH = ensureSlash(val)
		}
	}
	return nil
}

func applyEnv(cfg *Config) {
	envStr := map[string]*string{
		"GOPRESS_DB_NAME":          &cfg.DBName,
		"GOPRESS_DB_USER":          &cfg.DBUser,
		"GOPRESS_DB_PASSWORD":      &cfg.DBPassword,
		"GOPRESS_DB_HOST":          &cfg.DBHost,
		"GOPRESS_DB_CHARSET":       &cfg.DBCharset,
		"GOPRESS_TABLE_PREFIX":     &cfg.TablePrefix,
		"GOPRESS_AUTH_KEY":         &cfg.AuthKey,
		"GOPRESS_SECURE_AUTH_KEY":  &cfg.SecureAuthKey,
		"GOPRESS_LOGGED_IN_KEY":    &cfg.LoggedInKey,
		"GOPRESS_NONCE_KEY":        &cfg.NonceKey,
		"GOPRESS_AUTH_SALT":        &cfg.AuthSalt,
		"GOPRESS_SECURE_AUTH_SALT": &cfg.SecureAuthSalt,
		"GOPRESS_LOGGED_IN_SALT":   &cfg.LoggedInSalt,
		"GOPRESS_NONCE_SALT":       &cfg.NonceSalt,
		"GOPRESS_ENVIRONMENT_TYPE": &cfg.EnvironmentType,
		"WORDPRESS_DB_NAME":        &cfg.DBName,
		"WORDPRESS_DB_USER":        &cfg.DBUser,
		"WORDPRESS_DB_PASSWORD":    &cfg.DBPassword,
		"WORDPRESS_DB_HOST":        &cfg.DBHost,
	}
	for k, ptr := range envStr {
		if v, ok := os.LookupEnv(k); ok && v != "" {
			*ptr = v
		}
	}
	if v, ok := os.LookupEnv("GOPRESS_DEBUG"); ok {
		cfg.Debug = phpBool(v)
	}
}

func phpScalar(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if strings.EqualFold(raw, "true") {
		return "true", nil
	}
	if strings.EqualFold(raw, "false") {
		return "false", nil
	}
	if len(raw) >= 2 {
		q := raw[0]
		if (q == '\'' || q == '"') && raw[len(raw)-1] == q {
			inner := raw[1 : len(raw)-1]
			inner = strings.ReplaceAll(inner, `\'`, `'`)
			inner = strings.ReplaceAll(inner, `\"`, `"`)
			inner = strings.ReplaceAll(inner, `\\`, `\`)
			return inner, nil
		}
	}
	if _, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return raw, nil
	}
	return "", fmt.Errorf("unsupported scalar: %s", raw)
}

func phpBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func ensureSlash(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasSuffix(p, string(filepath.Separator)) && !strings.HasSuffix(p, "/") {
		return p + "/"
	}
	return p
}

func stripPHPComments(src string) string {
	// Block comments.
	reBlock := regexp.MustCompile(`(?s)/\*.*?\*/`)
	src = reBlock.ReplaceAllString(src, "")
	// Line comments.
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		if i := strings.Index(line, "#"); i >= 0 {
			// Avoid eating HTML/anchors inside strings — good enough for defines.
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
