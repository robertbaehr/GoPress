// Metadata API (P1.8) — port of wp-includes/meta.php core getters/setters.
//
// Uses *db.DB and ObjectCache groups "{type}_meta" (post_meta, user_meta, …).
// PHP serialization is not implemented yet; values are stored and returned as strings.
package wp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/db"
)

// MetaType is a WordPress metadata object type with an associated *meta table.
type MetaType string

const (
	MetaPost    MetaType = "post"
	MetaUser    MetaType = "user"
	MetaTerm    MetaType = "term"
	MetaComment MetaType = "comment"
)

// Meta is the WordPress metadata API bound to a DB and object cache.
type Meta struct {
	DB    *db.DB
	Cache *ObjectCache
	// Lazy optionally batch-primes meta caches on first GetMetadata (P2.17).
	Lazy *MetadataLazyloader
}

// NewMeta builds a Meta API. cache may be nil (lookups hit the DB only).
func NewMeta(database *db.DB, cache *ObjectCache) *Meta {
	if cache == nil {
		cache = NewObjectCache()
	}
	m := &Meta{DB: database, Cache: cache}
	m.Lazy = NewMetadataLazyloader(m)
	return m
}

// GetMetadata returns metadata for objectID.
//
// When key is empty, returns all meta as map[string][]string (or nil/empty).
// When single is true and key is set, returns the first value (string) or "".
// When single is false and key is set, returns []string (possibly empty).
func (m *Meta) GetMetadata(metaType MetaType, objectID int64, key string, single bool) any {
	if m == nil || objectID <= 0 || !validMetaType(metaType) {
		if single {
			return ""
		}
		return []string{}
	}
	// P2.17: flush queued IDs via update_meta_cache before reading.
	if m.Lazy != nil && m.Lazy.PendingCount(metaType) > 0 {
		m.Lazy.Flush(metaType, objectID)
	}

	cache := m.metaCache(metaType, objectID)
	if cache == nil {
		if single {
			return ""
		}
		if key == "" {
			return map[string][]string{}
		}
		return []string{}
	}

	if key == "" {
		return cache
	}

	vals, ok := cache[key]
	if !ok || len(vals) == 0 {
		if single {
			return ""
		}
		return []string{}
	}
	if single {
		return vals[0]
	}
	out := make([]string, len(vals))
	copy(out, vals)
	return out
}

// AddMetadata inserts a metadata row. When unique is true and the key already
// exists, returns (0, false). On success returns the new meta ID.
func (m *Meta) AddMetadata(metaType MetaType, objectID int64, key string, value any, unique bool) (int64, bool, error) {
	if m == nil || m.DB == nil || objectID <= 0 || key == "" || !validMetaType(metaType) {
		return 0, false, fmt.Errorf("wp: AddMetadata: invalid state")
	}
	table, column, err := m.tableInfo(metaType)
	if err != nil {
		return 0, false, err
	}
	serialized := metaValueString(value)

	if unique {
		q, args, err := m.DB.Prepare(
			"SELECT COUNT(*) FROM "+table+" WHERE meta_key = %s AND "+column+" = %d",
			key, objectID,
		)
		if err != nil {
			return 0, false, err
		}
		v, err := m.DB.GetVar(q, args...)
		if err != nil {
			return 0, false, err
		}
		if asInt64OrZero(v) > 0 {
			return 0, false, nil
		}
	}

	q, args, err := m.DB.Prepare(
		"INSERT INTO "+table+" ("+column+", meta_key, meta_value) VALUES (%d, %s, %s)",
		objectID, key, serialized,
	)
	if err != nil {
		return 0, false, err
	}
	res, err := m.DB.Exec(q, args...)
	if err != nil {
		return 0, false, err
	}
	mid, _ := res.LastInsertId()
	m.invalidate(metaType, objectID)
	return mid, true, nil
}

// UpdateMetadata updates existing meta for key, or adds it when missing.
// When prevValue is non-empty, only rows with that value are updated.
// Returns (metaID, added, updated, error). metaID is set when a row was added.
func (m *Meta) UpdateMetadata(metaType MetaType, objectID int64, key string, value any, prevValue string) (int64, bool, bool, error) {
	if m == nil || m.DB == nil || objectID <= 0 || key == "" || !validMetaType(metaType) {
		return 0, false, false, fmt.Errorf("wp: UpdateMetadata: invalid state")
	}
	table, column, err := m.tableInfo(metaType)
	if err != nil {
		return 0, false, false, err
	}
	idColumn := metaIDColumn(metaType)
	serialized := metaValueString(value)

	if prevValue == "" {
		old := m.GetMetadata(metaType, objectID, key, false)
		if vals, ok := old.([]string); ok && len(vals) == 1 && vals[0] == serialized {
			return 0, false, false, nil
		}
	}

	q, args, err := m.DB.Prepare(
		"SELECT "+idColumn+" FROM "+table+" WHERE meta_key = %s AND "+column+" = %d",
		key, objectID,
	)
	if err != nil {
		return 0, false, false, err
	}
	rows, err := m.DB.GetResults(q, args...)
	if err != nil {
		return 0, false, false, err
	}
	if len(rows) == 0 {
		mid, ok, err := m.AddMetadata(metaType, objectID, key, value, false)
		return mid, ok, false, err
	}

	var (
		uq    string
		uargs []any
	)
	if prevValue != "" {
		uq, uargs, err = m.DB.Prepare(
			"UPDATE "+table+" SET meta_value = %s WHERE meta_key = %s AND "+column+" = %d AND meta_value = %s",
			serialized, key, objectID, prevValue,
		)
	} else {
		uq, uargs, err = m.DB.Prepare(
			"UPDATE "+table+" SET meta_value = %s WHERE meta_key = %s AND "+column+" = %d",
			serialized, key, objectID,
		)
	}
	if err != nil {
		return 0, false, false, err
	}
	res, err := m.DB.Exec(uq, uargs...)
	if err != nil {
		return 0, false, false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, false, false, nil
	}
	m.invalidate(metaType, objectID)
	return 0, false, true, nil
}

// DeleteMetadata deletes metadata rows for key. When value is non-empty, only
// matching values are deleted. Returns true if at least one row was removed.
func (m *Meta) DeleteMetadata(metaType MetaType, objectID int64, key string, value string) (bool, error) {
	if m == nil || m.DB == nil || objectID <= 0 || key == "" || !validMetaType(metaType) {
		return false, fmt.Errorf("wp: DeleteMetadata: invalid state")
	}
	table, column, err := m.tableInfo(metaType)
	if err != nil {
		return false, err
	}

	var (
		q    string
		args []any
	)
	if value != "" {
		q, args, err = m.DB.Prepare(
			"DELETE FROM "+table+" WHERE meta_key = %s AND "+column+" = %d AND meta_value = %s",
			key, objectID, value,
		)
	} else {
		q, args, err = m.DB.Prepare(
			"DELETE FROM "+table+" WHERE meta_key = %s AND "+column+" = %d",
			key, objectID,
		)
	}
	if err != nil {
		return false, err
	}
	res, err := m.DB.Exec(q, args...)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return false, nil
	}
	m.invalidate(metaType, objectID)
	return true, nil
}

// UpdateMetaCache loads meta for objectIDs into the object cache.
func (m *Meta) UpdateMetaCache(metaType MetaType, objectIDs []int64) (map[int64]map[string][]string, error) {
	out := make(map[int64]map[string][]string, len(objectIDs))
	if m == nil || !validMetaType(metaType) || len(objectIDs) == 0 {
		return out, nil
	}

	group := metaCacheGroup(metaType)
	var missing []int64
	for _, id := range objectIDs {
		if id <= 0 {
			continue
		}
		if cached, ok := m.getCachedMeta(group, id); ok {
			out[id] = cached
			continue
		}
		missing = append(missing, id)
	}
	if len(missing) == 0 || m.DB == nil {
		for _, id := range missing {
			empty := map[string][]string{}
			out[id] = empty
			_ = m.Cache.Add(metaCacheKey(id), empty, group)
		}
		return out, nil
	}

	table, column, err := m.tableInfo(metaType)
	if err != nil {
		return nil, err
	}
	idColumn := metaIDColumn(metaType)

	placeholders := make([]string, len(missing))
	args := make([]any, len(missing))
	for i, id := range missing {
		placeholders[i] = "%d"
		args[i] = id
	}
	query := "SELECT " + column + ", meta_key, meta_value FROM " + table +
		" WHERE " + column + " IN (" + strings.Join(placeholders, ", ") + ")" +
		" ORDER BY " + idColumn + " ASC"
	q, prepArgs, err := m.DB.Prepare(query, args...)
	if err != nil {
		return nil, err
	}
	rows, err := m.DB.GetResults(q, prepArgs...)
	if err != nil {
		return nil, err
	}

	for _, id := range missing {
		out[id] = map[string][]string{}
	}
	for _, row := range rows {
		oid := asInt64OrZero(row[column])
		if oid == 0 {
			continue
		}
		mkey := metaValueString(row["meta_key"])
		mval := metaValueString(row["meta_value"])
		bucket := out[oid]
		if bucket == nil {
			bucket = map[string][]string{}
			out[oid] = bucket
		}
		bucket[mkey] = append(bucket[mkey], mval)
	}
	for _, id := range missing {
		_ = m.Cache.Add(metaCacheKey(id), out[id], group)
	}
	return out, nil
}

func (m *Meta) metaCache(metaType MetaType, objectID int64) map[string][]string {
	group := metaCacheGroup(metaType)
	if cached, ok := m.getCachedMeta(group, objectID); ok {
		return cached
	}
	loaded, err := m.UpdateMetaCache(metaType, []int64{objectID})
	if err != nil {
		return nil
	}
	return loaded[objectID]
}

func (m *Meta) getCachedMeta(group string, objectID int64) (map[string][]string, bool) {
	if m == nil || m.Cache == nil {
		return nil, false
	}
	v, ok := m.Cache.Get(metaCacheKey(objectID), group)
	if !ok {
		return nil, false
	}
	switch t := v.(type) {
	case map[string][]string:
		return t, true
	case map[string]any:
		out := make(map[string][]string, len(t))
		for k, val := range t {
			switch vv := val.(type) {
			case []string:
				out[k] = vv
			case []any:
				ss := make([]string, len(vv))
				for i, x := range vv {
					ss[i] = metaValueString(x)
				}
				out[k] = ss
			default:
				out[k] = []string{metaValueString(vv)}
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func (m *Meta) invalidate(metaType MetaType, objectID int64) {
	if m == nil || m.Cache == nil {
		return
	}
	m.Cache.Delete(metaCacheKey(objectID), metaCacheGroup(metaType))
}

func (m *Meta) tableInfo(metaType MetaType) (table, column string, err error) {
	if m == nil || m.DB == nil {
		return "", "", fmt.Errorf("wp: meta: nil DB")
	}
	switch metaType {
	case MetaPost:
		return m.DB.PostMeta(), "post_id", nil
	case MetaUser:
		return m.DB.UserMeta(), "user_id", nil
	case MetaTerm:
		return m.DB.TermMeta(), "term_id", nil
	case MetaComment:
		return m.DB.CommentMeta(), "comment_id", nil
	default:
		return "", "", fmt.Errorf("wp: meta: unknown type %q", metaType)
	}
}

func validMetaType(t MetaType) bool {
	switch t {
	case MetaPost, MetaUser, MetaTerm, MetaComment:
		return true
	default:
		return false
	}
}

func metaCacheGroup(t MetaType) string { return string(t) + "_meta" }

func metaCacheKey(objectID int64) string { return strconv.FormatInt(objectID, 10) }

func metaIDColumn(t MetaType) string {
	if t == MetaUser {
		return "umeta_id"
	}
	return "meta_id"
}

func metaValueString(v any) string {
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

func asInt64OrZero(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case uint64:
		return int64(n)
	case float64:
		return int64(n)
	case []byte:
		i, _ := strconv.ParseInt(string(n), 10, 64)
		return i
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	default:
		return 0
	}
}
