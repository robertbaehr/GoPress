package wp

import (
	"net/url"
	"strings"
)

// UTF8URIEncode mirrors utf8_uri_encode (P2.144).
// Encodes non-ASCII UTF-8 octets as %XX; optionally percent-encodes ASCII via
// rawurlencode when encodeASCII is true. length limits the output byte length
// (0 = unlimited), matching PHP strlen counting of the encoded result.
func UTF8URIEncode(s string, length int, encodeASCII bool) string {
	return utf8URIEncodeOpts(s, length, encodeASCII)
}

// WPUTF8URIEncode is an alias of UTF8URIEncode (P2.144).
func WPUTF8URIEncode(s string, length int, encodeASCII bool) string {
	return UTF8URIEncode(s, length, encodeASCII)
}

// utf8URIEncode is the historical helper used by sanitize_title_with_dashes
// (encodeASCII=false). Kept for call-site compatibility.
func utf8URIEncode(s string, length int) string {
	return utf8URIEncodeOpts(s, length, false)
}

// utf8URIEncodeOpts ports utf8_uri_encode octet scanning (P2.144).
// PHP walks bytes and groups multi-byte sequences by lead-byte class before
// emitting %xx for each octet (dechex, lowercase).
func utf8URIEncodeOpts(s string, length int, encodeASCII bool) string {
	var b strings.Builder
	n := 0
	i := 0
	for i < len(s) {
		value := s[i]
		if value < 128 {
			char := string([]byte{value})
			encoded := char
			if encodeASCII {
				encoded = url.QueryEscape(char)
				// rawurlencode uses %20 for space; QueryEscape uses +.
				encoded = strings.ReplaceAll(encoded, "+", "%20")
			}
			encLen := len(encoded)
			if length > 0 && n+encLen > length {
				break
			}
			b.WriteString(encoded)
			n += encLen
			i++
			continue
		}
		// Multi-byte sequence length from lead byte (PHP utf8_uri_encode).
		numOctets := 1
		if value < 224 {
			numOctets = 2
		} else if value < 240 {
			numOctets = 3
		} else {
			numOctets = 4
		}
		if i+numOctets > len(s) {
			// Truncated sequence — encode remaining bytes individually.
			numOctets = len(s) - i
		}
		need := numOctets * 3
		if length > 0 && n+need > length {
			break
		}
		for j := 0; j < numOctets; j++ {
			b.WriteString("%" + strings.ToLower(hexByte(s[i+j])))
		}
		n += need
		i += numOctets
	}
	return b.String()
}
