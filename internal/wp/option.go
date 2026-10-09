// Options API (P1.7) — port of wp-includes/option.php core getters/setters.
//
// Uses *db.DB and ObjectCache group "options" (alloptions / notoptions / per-key).
// PHP serialization is not implemented yet; values are stored and returned as strings.
package wp

import (
	"fmt"
	"strings"

	"github.com/robertbaehr/GoPress/internal/db"
)

// Autoload values that WordPress loads at boot (wp_autoload_values_to_autoload).
var autoloadValues = []string{"yes", "on", "auto-on", "auto"}

// Options is the WordPress options API bound to a DB and object cache.
type Options struct {
	DB    *db.DB
	Cache *ObjectCache
}

// NewOptions builds an Options API. cache may be nil (lookups hit the DB only).
func NewOptions(database *db.DB, cache *ObjectCache) *Options {
	if cache == nil {
		cache = NewObjectCache()
	}
	return &Options{DB: database, Cache: cache}
}

// GetOption returns the option value, or defaultValue when missing / on error.
// Mirrors get_option(); missing options are remembered in the notoptions cache.
func (o *Options) GetOption(name string, defaultValue any) any {
	name = strings.TrimSpace(name)
	if o == nil || name == "" {
		return defaultValue
	}

	if all, ok := o.alloptions(); ok {
		if v, exists := all[name]; exists {
			return v
		}
	}

	if o.isNotOption(name) {
		return defaultValue
	}

	if v, ok := o.Cache.Get(name, "options"); ok {
		return v
	}

	if o.DB == nil {
		return defaultValue
	}

	q, args, err := o.DB.Prepare(
		"SELECT option_value FROM "+o.DB.Options()+" WHERE option_name = %s LIMIT 1",
		name,
	)
	if err != nil {
		return defaultValue
	}
	row, err := o.DB.GetRow(q, args...)
	if err != nil || row == nil {
		o.rememberNotOption(name)
		return defaultValue
	}
	val := optionValueString(row["option_value"])
	_ = o.Cache.Add(name, val, "options")
	return val
}

// AddOption inserts a new option. Returns false if it already exists.
// autoload: empty/"yes"/"on"/"auto" → "on"; "no"/"off" → "off".
func (o *Options) AddOption(name string, value any, autoload string) (bool, error) {
	name = strings.TrimSpace(name)
	if o == nil || o.DB == nil || name == "" {
		return false, fmt.Errorf("wp: AddOption: invalid state")
	}
	if err := protectSpecialOption(name); err != nil {
		return false, err
	}

	if !o.isNotOption(name) {
		if cur := o.GetOption(name, nil); cur != nil {
			return false, nil
		}
	}

	serialized := optionValueString(value)
	al := normalizeAutoload(autoload)

	q, args, err := o.DB.Prepare(
		"INSERT INTO "+o.DB.Options()+" (option_name, option_value, autoload) VALUES (%s, %s, %s)",
		name, serialized, al,
	)
	if err != nil {
		return false, err
	}
	if _, err := o.DB.Exec(q, args...); err != nil {
		return false, err
	}

	o.clearNotOption(name)
	if isAutoload(al) {
		all, _ := o.alloptions()
		if all == nil {
			all = map[string]string{}
		}
		all[name] = serialized
		o.Cache.Set("alloptions", all, "options")
		o.Cache.Delete(name, "options")
	} else {
		o.Cache.Set(name, serialized, "options")
		if all, ok := o.alloptions(); ok {
			delete(all, name)
			o.Cache.Set("alloptions", all, "options")
		}
	}
	return true, nil
}

// UpdateOption updates an existing option, or adds it when missing.
// Pass autoload "" to leave the existing autoload column unchanged on update.
func (o *Options) UpdateOption(name string, value any, autoload string) (bool, error) {
	name = strings.TrimSpace(name)
	if o == nil || o.DB == nil || name == "" {
		return false, fmt.Errorf("wp: UpdateOption: invalid state")
	}
	if err := protectSpecialOption(name); err != nil {
		return false, err
	}

	serialized := optionValueString(value)
	old := o.GetOption(name, nil)
	if old != nil && optionValueString(old) == serialized {
		return false, nil
	}
	if old == nil {
		return o.AddOption(name, value, autoload)
	}

	var (
		q    string
		args []any
		err  error
	)
	if autoload != "" {
		al := normalizeAutoload(autoload)
		q, args, err = o.DB.Prepare(
			"UPDATE "+o.DB.Options()+" SET option_value = %s, autoload = %s WHERE option_name = %s",
			serialized, al, name,
		)
	} else {
		q, args, err = o.DB.Prepare(
			"UPDATE "+o.DB.Options()+" SET option_value = %s WHERE option_name = %s",
			serialized, name,
		)
	}
	if err != nil {
		return false, err
	}
	res, err := o.DB.Exec(q, args...)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return false, nil
	}

	o.clearNotOption(name)

	// Refresh caches: prefer alloptions when the option is autoloaded.
	al := autoload
	if al == "" {
		al = o.autoloadOf(name)
	} else {
		al = normalizeAutoload(al)
	}
	if isAutoload(al) {
		o.Cache.Delete(name, "options")
		all, _ := o.alloptions()
		if all == nil {
			all = map[string]string{}
		}
		all[name] = serialized
		o.Cache.Set("alloptions", all, "options")
	} else {
		if all, ok := o.alloptions(); ok {
			delete(all, name)
			o.Cache.Set("alloptions", all, "options")
		}
		o.Cache.Set(name, serialized, "options")
	}
	return true, nil
}

// DeleteOption removes an option. Returns false if it did not exist.
func (o *Options) DeleteOption(name string) (bool, error) {
	name = strings.TrimSpace(name)
	if o == nil || o.DB == nil || name == "" {
		return false, fmt.Errorf("wp: DeleteOption: invalid state")
	}
	if err := protectSpecialOption(name); err != nil {
		return false, err
	}

	al := o.autoloadOf(name)
	q, args, err := o.DB.Prepare(
		"DELETE FROM "+o.DB.Options()+" WHERE option_name = %s",
		name,
	)
	if err != nil {
		return false, err
	}
	res, err := o.DB.Exec(q, args...)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return false, nil
	}

	if isAutoload(al) {
		if all, ok := o.alloptions(); ok {
			delete(all, name)
			o.Cache.Set("alloptions", all, "options")
		}
	} else {
		o.Cache.Delete(name, "options")
	}
	o.rememberNotOption(name)
	return true, nil
}

// LoadAutoload loads autoloaded options into the alloptions cache (wp_load_alloptions).
func (o *Options) LoadAutoload() error {
	if o == nil || o.DB == nil {
		return fmt.Errorf("wp: LoadAutoload: nil Options or DB")
	}
	if _, ok := o.Cache.Get("alloptions", "options"); ok {
		return nil
	}

	placeholders := make([]string, len(autoloadValues))
	args := make([]any, len(autoloadValues))
	for i, v := range autoloadValues {
		placeholders[i] = "%s"
		args[i] = v
	}
	query := "SELECT option_name, option_value FROM " + o.DB.Options() +
		" WHERE autoload IN (" + strings.Join(placeholders, ", ") + ")"
	q, prepArgs, err := o.DB.Prepare(query, args...)
	if err != nil {
		return err
	}
	rows, err := o.DB.GetResults(q, prepArgs...)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		// Fallback: load everything (WordPress does this when autoload query is empty).
		q2, a2, err := o.DB.Prepare("SELECT option_name, option_value FROM " + o.DB.Options())
		if err != nil {
			return err
		}
		rows, err = o.DB.GetResults(q2, a2...)
		if err != nil {
			return err
		}
	}

	all := make(map[string]string, len(rows))
	for _, row := range rows {
		name := optionValueString(row["option_name"])
		all[name] = optionValueString(row["option_value"])
	}
	o.Cache.Add("alloptions", all, "options")
	return nil
}

func (o *Options) alloptions() (map[string]string, bool) {
	if o == nil || o.Cache == nil {
		return nil, false
	}
	v, ok := o.Cache.Get("alloptions", "options")
	if !ok {
		return nil, false
	}
	switch m := v.(type) {
	case map[string]string:
		return m, true
	case map[string]any:
		out := make(map[string]string, len(m))
		for k, val := range m {
			out[k] = optionValueString(val)
		}
		return out, true
	default:
		return nil, false
	}
}

func (o *Options) isNotOption(name string) bool {
	if o == nil || o.Cache == nil {
		return false
	}
	v, ok := o.Cache.Get("notoptions", "options")
	if !ok {
		return false
	}
	switch m := v.(type) {
	case map[string]bool:
		return m[name]
	case map[string]any:
		_, exists := m[name]
		return exists
	default:
		return false
	}
}

func (o *Options) rememberNotOption(name string) {
	if o == nil || o.Cache == nil {
		return
	}
	not := map[string]bool{}
	if v, ok := o.Cache.Get("notoptions", "options"); ok {
		if m, ok := v.(map[string]bool); ok {
			not = m
		}
	}
	not[name] = true
	o.Cache.Set("notoptions", not, "options")
}

func (o *Options) clearNotOption(name string) {
	if o == nil || o.Cache == nil {
		return
	}
	v, ok := o.Cache.Get("notoptions", "options")
	if !ok {
		return
	}
	if m, ok := v.(map[string]bool); ok {
		delete(m, name)
		o.Cache.Set("notoptions", m, "options")
	}
}

func (o *Options) autoloadOf(name string) string {
	if o == nil || o.DB == nil {
		return ""
	}
	q, args, err := o.DB.Prepare(
		"SELECT autoload FROM "+o.DB.Options()+" WHERE option_name = %s LIMIT 1",
		name,
	)
	if err != nil {
		return ""
	}
	v, err := o.DB.GetVar(q, args...)
	if err != nil || v == nil {
		return ""
	}
	return optionValueString(v)
}

func protectSpecialOption(name string) error {
	if name == "alloptions" || name == "notoptions" {
		return fmt.Errorf("wp: protected option %q", name)
	}
	return nil
}

func optionValueString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case bool:
		if t {
			return "1"
		}
		return ""
	default:
		return fmt.Sprint(t)
	}
}

func normalizeAutoload(autoload string) string {
	switch strings.ToLower(strings.TrimSpace(autoload)) {
	case "", "yes", "on", "true", "1", "auto", "auto-on":
		return "on"
	case "no", "off", "false", "0", "auto-off":
		return "off"
	default:
		return "on"
	}
}

func isAutoload(autoload string) bool {
	al := strings.ToLower(strings.TrimSpace(autoload))
	for _, v := range autoloadValues {
		if al == v {
			return true
		}
	}
	return false
}
