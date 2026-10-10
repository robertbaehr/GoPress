// Minimal PHP serialize/unserialize for WordPress option and usermeta values.
// Supports arrays, strings, ints, bools, null, and floats (subset of PHP serialize).
package wp

import (
	"fmt"
	"strconv"
	"strings"
)

// phpUnserialize parses a PHP-serialized value into Go types:
//
//	array  → map[any]any (string keys preserved; int keys as int)
//	string → string
//	int    → int64
//	bool   → bool
//	null   → nil
//	float  → float64
func phpUnserialize(data string) (any, error) {
	v, _, err := phpParse(data, 0)
	return v, err
}

func phpParse(data string, i int) (any, int, error) {
	if i >= len(data) {
		return nil, i, fmt.Errorf("php unserialize: unexpected end")
	}
	switch data[i] {
	case 'N':
		if i+1 >= len(data) || data[i+1] != ';' {
			return nil, i, fmt.Errorf("php unserialize: bad null")
		}
		return nil, i + 2, nil
	case 'b':
		if i+3 >= len(data) || data[i+1] != ':' || data[i+3] != ';' {
			return nil, i, fmt.Errorf("php unserialize: bad bool")
		}
		return data[i+2] == '1', i + 4, nil
	case 'i':
		j := i + 2
		if i+1 >= len(data) || data[i+1] != ':' {
			return nil, i, fmt.Errorf("php unserialize: bad int")
		}
		k := strings.IndexByte(data[j:], ';')
		if k < 0 {
			return nil, i, fmt.Errorf("php unserialize: bad int terminator")
		}
		n, err := strconv.ParseInt(data[j:j+k], 10, 64)
		if err != nil {
			return nil, i, err
		}
		return n, j + k + 1, nil
	case 'd':
		j := i + 2
		if i+1 >= len(data) || data[i+1] != ':' {
			return nil, i, fmt.Errorf("php unserialize: bad float")
		}
		k := strings.IndexByte(data[j:], ';')
		if k < 0 {
			return nil, i, fmt.Errorf("php unserialize: bad float terminator")
		}
		f, err := strconv.ParseFloat(data[j:j+k], 64)
		if err != nil {
			return nil, i, err
		}
		return f, j + k + 1, nil
	case 's':
		// s:N:"…";
		if i+1 >= len(data) || data[i+1] != ':' {
			return nil, i, fmt.Errorf("php unserialize: bad string")
		}
		j := i + 2
		colon := strings.IndexByte(data[j:], ':')
		if colon < 0 {
			return nil, i, fmt.Errorf("php unserialize: bad string length")
		}
		n, err := strconv.Atoi(data[j : j+colon])
		if err != nil {
			return nil, i, err
		}
		j += colon + 1
		if j >= len(data) || data[j] != '"' {
			return nil, i, fmt.Errorf("php unserialize: bad string open quote")
		}
		j++
		if j+n > len(data) {
			return nil, i, fmt.Errorf("php unserialize: string overrun")
		}
		s := data[j : j+n]
		j += n
		if j+1 >= len(data) || data[j] != '"' || data[j+1] != ';' {
			return nil, i, fmt.Errorf("php unserialize: bad string close")
		}
		return s, j + 2, nil
	case 'a':
		// a:N:{…}
		if i+1 >= len(data) || data[i+1] != ':' {
			return nil, i, fmt.Errorf("php unserialize: bad array")
		}
		j := i + 2
		colon := strings.IndexByte(data[j:], ':')
		if colon < 0 {
			return nil, i, fmt.Errorf("php unserialize: bad array length")
		}
		n, err := strconv.Atoi(data[j : j+colon])
		if err != nil {
			return nil, i, err
		}
		j += colon + 1
		if j >= len(data) || data[j] != '{' {
			return nil, i, fmt.Errorf("php unserialize: bad array open")
		}
		j++
		m := make(map[any]any, n)
		for k := 0; k < n; k++ {
			key, nj, err := phpParse(data, j)
			if err != nil {
				return nil, i, err
			}
			j = nj
			val, nj, err := phpParse(data, j)
			if err != nil {
				return nil, i, err
			}
			j = nj
			m[key] = val
		}
		if j >= len(data) || data[j] != '}' {
			return nil, i, fmt.Errorf("php unserialize: bad array close")
		}
		return m, j + 1, nil
	default:
		return nil, i, fmt.Errorf("php unserialize: unsupported type %q at %d", data[i], i)
	}
}

// phpMapStringBool converts a PHP array map into map[string]bool.
func phpMapStringBool(v any) map[string]bool {
	m, ok := v.(map[any]any)
	if !ok || m == nil {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(m))
	for k, val := range m {
		ks := fmt.Sprint(k)
		switch t := val.(type) {
		case bool:
			out[ks] = t
		case int64:
			out[ks] = t != 0
		case string:
			out[ks] = t != "" && t != "0"
		default:
			out[ks] = val != nil
		}
	}
	return out
}

// phpSerialize encodes v as PHP serialize (arrays, strings, ints, bools, null, floats).
func phpSerialize(v any) (string, error) {
	var b strings.Builder
	if err := phpWrite(&b, v); err != nil {
		return "", err
	}
	return b.String(), nil
}

func phpWrite(b *strings.Builder, v any) error {
	switch t := v.(type) {
	case nil:
		b.WriteString("N;")
		return nil
	case bool:
		if t {
			b.WriteString("b:1;")
		} else {
			b.WriteString("b:0;")
		}
		return nil
	case int:
		b.WriteString("i:")
		b.WriteString(strconv.FormatInt(int64(t), 10))
		b.WriteByte(';')
		return nil
	case int64:
		b.WriteString("i:")
		b.WriteString(strconv.FormatInt(t, 10))
		b.WriteByte(';')
		return nil
	case float64:
		b.WriteString("d:")
		b.WriteString(strconv.FormatFloat(t, 'g', -1, 64))
		b.WriteByte(';')
		return nil
	case string:
		b.WriteString("s:")
		b.WriteString(strconv.Itoa(len(t)))
		b.WriteString(":\"")
		b.WriteString(t)
		b.WriteString("\";")
		return nil
	case map[string]any:
		b.WriteString("a:")
		b.WriteString(strconv.Itoa(len(t)))
		b.WriteString(":{")
		for k, val := range t {
			if err := phpWrite(b, k); err != nil {
				return err
			}
			if err := phpWrite(b, val); err != nil {
				return err
			}
		}
		b.WriteByte('}')
		return nil
	case map[any]any:
		b.WriteString("a:")
		b.WriteString(strconv.Itoa(len(t)))
		b.WriteString(":{")
		for k, val := range t {
			if err := phpWrite(b, k); err != nil {
				return err
			}
			if err := phpWrite(b, val); err != nil {
				return err
			}
		}
		b.WriteByte('}')
		return nil
	default:
		return fmt.Errorf("php serialize: unsupported type %T", v)
	}
}
