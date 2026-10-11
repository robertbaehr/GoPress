package wp

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

// ExifFrac2Dec converts an EXIF fraction string "n/d" to float (wp_exif_frac2dec, P8.33).
func ExifFrac2Dec(str string) float64 {
	str = strings.TrimSpace(str)
	if str == "" {
		return 0
	}
	if strings.Count(str, "/") != 1 {
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return f
		}
		return 0
	}
	parts := strings.SplitN(str, "/", 2)
	num, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	den, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil || den == 0 {
		return 0
	}
	return num / den
}

// ExifDate2TS converts EXIF "Y:m:d H:i:s" to a unix timestamp (wp_exif_date2ts, P8.33).
// Returns 0 on failure (PHP returns false; Go callers treat 0 as empty).
func ExifDate2TS(str string) int64 {
	str = strings.TrimSpace(str)
	parts := strings.SplitN(str, " ", 2)
	if len(parts) != 2 {
		return 0
	}
	d := strings.Split(parts[0], ":")
	if len(d) != 3 {
		return 0
	}
	// Prefer local parse like strtotime of "Y-m-d H:i:s".
	layout := "2006-01-02 15:04:05"
	ts, err := time.ParseInLocation(layout, d[0]+"-"+d[1]+"-"+d[2]+" "+parts[1], time.Local)
	if err != nil {
		return 0
	}
	return ts.Unix()
}

// applyJPEGExif fills image_meta fields from JPEG EXIF (P8.33).
func applyJPEGExif(meta map[string]any, file string) {
	exif, ok := readJPEGExif(file)
	if !ok || len(exif) == 0 {
		return
	}
	desc := strings.TrimSpace(exif["ImageDescription"])
	if desc != "" {
		if metaString(meta, "title") == "" && len(desc) < 80 {
			meta["title"] = desc
		}
		if metaString(meta, "caption") == "" {
			meta["caption"] = desc
		}
	}
	if metaString(meta, "credit") == "" {
		if artist := strings.TrimSpace(exif["Artist"]); artist != "" {
			meta["credit"] = artist
		}
	}
	if metaString(meta, "copyright") == "" {
		if c := strings.TrimSpace(exif["Copyright"]); c != "" {
			meta["copyright"] = c
		}
	}
	if fn := strings.TrimSpace(exif["FNumber"]); fn != "" {
		meta["aperture"] = math.Round(ExifFrac2Dec(fn)*100) / 100
	}
	if model := strings.TrimSpace(exif["Model"]); model != "" {
		meta["camera"] = model
	}
	if metaInt64(meta, "created_timestamp") == 0 {
		if dt := strings.TrimSpace(exif["DateTimeDigitized"]); dt != "" {
			if ts := ExifDate2TS(dt); ts > 0 {
				meta["created_timestamp"] = ts
			}
		}
		if metaInt64(meta, "created_timestamp") == 0 {
			if dt := strings.TrimSpace(exif["DateTimeOriginal"]); dt != "" {
				if ts := ExifDate2TS(dt); ts > 0 {
					meta["created_timestamp"] = ts
				}
			}
		}
	}
	if fl := strings.TrimSpace(exif["FocalLength"]); fl != "" {
		meta["focal_length"] = ExifFrac2Dec(fl)
	}
	if iso := strings.TrimSpace(exif["ISOSpeedRatings"]); iso != "" {
		if f, err := strconv.ParseFloat(iso, 64); err == nil {
			meta["iso"] = f
		}
	}
	if et := strings.TrimSpace(exif["ExposureTime"]); et != "" {
		meta["shutter_speed"] = ExifFrac2Dec(et)
	}
	if ori := strings.TrimSpace(exif["Orientation"]); ori != "" {
		if n, err := strconv.Atoi(ori); err == nil {
			meta["orientation"] = n
		}
	}
}

func metaString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func metaInt64(m map[string]any, key string) int64 {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	default:
		return 0
	}
}

// readJPEGExif extracts a small set of EXIF tags from a JPEG file (P8.33).
// Returns nil,false when the file has no parseable Exif APP1 segment.
func readJPEGExif(file string) (map[string]string, bool) {
	data, err := os.ReadFile(file)
	if err != nil || len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil, false
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			i++
			continue
		}
		for i < len(data) && data[i] == 0xFF {
			i++
		}
		if i >= len(data) {
			break
		}
		marker := data[i]
		i++
		if marker == 0xD9 || marker == 0xDA { // EOI / SOS
			break
		}
		if i+2 > len(data) {
			break
		}
		segLen := int(binary.BigEndian.Uint16(data[i : i+2]))
		i += 2
		if segLen < 2 || i+segLen-2 > len(data) {
			break
		}
		payload := data[i : i+segLen-2]
		i += segLen - 2
		if marker != 0xE1 { // APP1
			continue
		}
		if len(payload) < 6 || string(payload[:4]) != "Exif" || payload[4] != 0 || payload[5] != 0 {
			continue
		}
		tags := parseTIFFExif(payload[6:])
		if len(tags) == 0 {
			return nil, false
		}
		return tags, true
	}
	return nil, false
}

func parseTIFFExif(tiff []byte) map[string]string {
	if len(tiff) < 8 {
		return nil
	}
	var order binary.ByteOrder
	switch string(tiff[0:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return nil
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return nil
	}
	ifd0 := int(order.Uint32(tiff[4:8]))
	out := map[string]string{}
	readIFD(tiff, order, ifd0, out, false)
	if ptr, ok := out["__ExifIFD"]; ok {
		delete(out, "__ExifIFD")
		off, err := strconv.Atoi(ptr)
		if err == nil && off > 0 {
			readIFD(tiff, order, off, out, true)
		}
	}
	return out
}

func readIFD(tiff []byte, order binary.ByteOrder, offset int, out map[string]string, exifIFD bool) {
	if offset < 0 || offset+2 > len(tiff) {
		return
	}
	count := int(order.Uint16(tiff[offset : offset+2]))
	pos := offset + 2
	for i := 0; i < count; i++ {
		if pos+12 > len(tiff) {
			return
		}
		tag := order.Uint16(tiff[pos : pos+2])
		typ := order.Uint16(tiff[pos+2 : pos+4])
		cnt := order.Uint32(tiff[pos+4 : pos+8])
		valOff := tiff[pos+8 : pos+12]
		pos += 12
		name, ok := exifTagName(tag, exifIFD)
		if !ok {
			// Exif IFD pointer lives in IFD0.
			if !exifIFD && tag == 0x8769 && typ == 4 && cnt == 1 {
				out["__ExifIFD"] = strconv.FormatUint(uint64(order.Uint32(valOff)), 10)
			}
			continue
		}
		if s, ok := decodeTIFFValue(tiff, order, typ, cnt, valOff); ok {
			out[name] = s
		}
	}
}

func exifTagName(tag uint16, exifIFD bool) (string, bool) {
	if !exifIFD {
		switch tag {
		case 0x010E:
			return "ImageDescription", true
		case 0x0110:
			return "Model", true
		case 0x0112:
			return "Orientation", true
		case 0x0132:
			return "DateTime", true
		case 0x013B:
			return "Artist", true
		case 0x8298:
			return "Copyright", true
		}
		return "", false
	}
	switch tag {
	case 0x829A:
		return "ExposureTime", true
	case 0x829D:
		return "FNumber", true
	case 0x8827:
		return "ISOSpeedRatings", true
	case 0x9003:
		return "DateTimeOriginal", true
	case 0x9004:
		return "DateTimeDigitized", true
	case 0x920A:
		return "FocalLength", true
	}
	return "", false
}

func decodeTIFFValue(tiff []byte, order binary.ByteOrder, typ uint16, cnt uint32, valOff []byte) (string, bool) {
	typeSize := tiffTypeSize(typ)
	if typeSize == 0 || cnt == 0 {
		return "", false
	}
	nbytes := int(cnt) * typeSize
	var raw []byte
	if nbytes <= 4 {
		raw = valOff[:nbytes]
	} else {
		off := int(order.Uint32(valOff))
		if off < 0 || off+nbytes > len(tiff) {
			return "", false
		}
		raw = tiff[off : off+nbytes]
	}
	switch typ {
	case 2: // ASCII
		s := string(raw)
		if i := strings.IndexByte(s, 0); i >= 0 {
			s = s[:i]
		}
		return strings.TrimSpace(s), true
	case 3: // SHORT
		if len(raw) < 2 {
			return "", false
		}
		return strconv.FormatUint(uint64(order.Uint16(raw[:2])), 10), true
	case 4: // LONG
		if len(raw) < 4 {
			return "", false
		}
		return strconv.FormatUint(uint64(order.Uint32(raw[:4])), 10), true
	case 5: // RATIONAL
		if len(raw) < 8 {
			return "", false
		}
		num := order.Uint32(raw[0:4])
		den := order.Uint32(raw[4:8])
		return strconv.FormatUint(uint64(num), 10) + "/" + strconv.FormatUint(uint64(den), 10), true
	default:
		return "", false
	}
}

func tiffTypeSize(typ uint16) int {
	switch typ {
	case 1, 2: // BYTE, ASCII
		return 1
	case 3: // SHORT
		return 2
	case 4: // LONG
		return 4
	case 5: // RATIONAL
		return 8
	default:
		return 0
	}
}
