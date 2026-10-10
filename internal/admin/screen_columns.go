package admin

import (
	"fmt"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// editColumn is a toggleable list-table column on edit.php (P7.80).
type editColumn struct {
	ID    string
	Label string
}

// editToggleColumns returns Screen Options Columns for edit-{post|page} (P7.80).
// Title / checkbox columns stay special and are not toggleable (WP_Screen).
func editToggleColumns(postType string) []editColumn {
	cols := []editColumn{
		{ID: "author", Label: "Author"},
		{ID: "date", Label: "Date"},
		{ID: "status", Label: "Status"},
	}
	if postType == "page" {
		// Insert Order after Author to match list-table header order.
		cols = []editColumn{
			{ID: "author", Label: "Author"},
			{ID: "menu_order", Label: "Order"},
			{ID: "date", Label: "Date"},
			{ID: "status", Label: "Status"},
		}
	}
	return cols
}

func editColumnsHiddenOption(postType string) string {
	if postType == "" {
		postType = "post"
	}
	return "manageedit-" + postType + "columnshidden"
}

// getEditHiddenColumns reads manageedit-{type}columnshidden (P7.80).
func getEditHiddenColumns(c *wp.Context, postType string) []string {
	raw := strings.TrimSpace(wp.GetUserOption(c, editColumnsHiddenOption(postType), 0))
	if raw == "" {
		return nil
	}
	v, err := wp.UnserializePHP(raw)
	if err != nil {
		// Comma-separated fallback (ajax-shaped).
		var out []string
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		return out
	}
	return phpArrayStrings(v)
}

func saveEditHiddenColumns(c *wp.Context, userID int64, postType string, hidden []string) error {
	if c == nil || userID <= 0 {
		return fmt.Errorf("admin: save hidden columns: invalid state")
	}
	arr := map[any]any{}
	for i, col := range hidden {
		arr[i] = col
	}
	serialized, err := wp.SerializePHP(arr)
	if err != nil {
		return err
	}
	return wp.UpdateUserOption(c, userID, editColumnsHiddenOption(postType), serialized)
}

func phpArrayStrings(v any) []string {
	m, ok := v.(map[any]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(m))
	for i := 0; i < len(m); i++ {
		val, ok := m[i]
		if !ok {
			val, ok = m[int64(i)]
		}
		if !ok {
			continue
		}
		s, ok := val.(string)
		if !ok {
			s = strings.TrimSpace(fmt.Sprint(val))
		}
		s = strings.TrimSpace(s)
		if s != "" && s != "<nil>" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		// Unordered fallback.
		for _, val := range m {
			s, ok := val.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func hiddenColumnSet(hidden []string) map[string]bool {
	out := map[string]bool{}
	for _, h := range hidden {
		out[h] = true
	}
	return out
}

func columnHiddenClass(hidden map[string]bool, id string) string {
	if hidden[id] {
		return " hidden"
	}
	return ""
}
