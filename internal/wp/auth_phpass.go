package wp

import (
	"crypto/md5"
	"strings"
)

// Minimal phpass portable hasher (public domain algorithm) for CheckPassword / HashPasswordPortable.
const phpassItoa64 = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func phpassEncode64(input []byte, count int) string {
	var out strings.Builder
	i := 0
	for {
		value := int(input[i])
		i++
		out.WriteByte(phpassItoa64[value&0x3f])
		if i < count {
			value |= int(input[i]) << 8
		}
		out.WriteByte(phpassItoa64[(value>>6)&0x3f])
		if i >= count {
			break
		}
		i++
		if i < count {
			value |= int(input[i]) << 16
		}
		out.WriteByte(phpassItoa64[(value>>12)&0x3f])
		if i >= count {
			break
		}
		i++
		out.WriteByte(phpassItoa64[(value>>18)&0x3f])
		if i >= count {
			break
		}
	}
	return out.String()
}

func phpassCryptPrivate(password, setting string) string {
	if len(setting) < 12 {
		return "*0"
	}
	id := setting[:3]
	if id != "$P$" && id != "$H$" {
		return "*0"
	}
	countLog2 := strings.IndexByte(phpassItoa64, setting[3])
	if countLog2 < 7 || countLog2 > 30 {
		return "*0"
	}
	count := 1 << countLog2
	salt := setting[4:12]
	if len(salt) != 8 {
		return "*0"
	}
	hash := md5.Sum([]byte(salt + password))
	for i := 0; i < count; i++ {
		hash = md5.Sum(append(hash[:], password...))
	}
	return setting[:12] + phpassEncode64(hash[:], 16)
}

func phpassCheck(password, stored string) bool {
	if len(stored) != 34 {
		return false
	}
	return phpassCryptPrivate(password, stored) == stored
}

// HashPasswordPortable creates a phpass portable hash ($P$) for tests / legacy.
func HashPasswordPortable(password string, iterationLog2 int) string {
	if iterationLog2 < 4 || iterationLog2 > 31 {
		iterationLog2 = 8
	}
	// Fixed salt material for determinism in tests when random is not needed;
	// for real use, callers should prefer HashPassword (bcrypt).
	saltBytes := md5.Sum([]byte("gopress-phpass-salt-" + password))
	idx := iterationLog2 + 5
	if idx > 30 {
		idx = 30
	}
	setting := "$P$" + string(phpassItoa64[idx]) + phpassEncode64(saltBytes[:], 6)
	return phpassCryptPrivate(password, setting)
}
