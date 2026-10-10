package wp

import (
	"crypto/rand"
	"math/big"
)

// maxRandomNumber mirrors WP's $max_random_number (0xffffffff).
const maxRandomNumber int64 = 4294967295

// Rand returns a cryptographically secure random int in [min, max]
// (wp_rand, P2.103). Arguments may be in either order; missing bounds
// default to 0 and 4294967295. Values are treated as ints (truncated).
func Rand(minMax ...int64) int64 {
	minV := int64(0)
	maxV := maxRandomNumber
	if len(minMax) >= 1 {
		minV = minMax[0]
	}
	if len(minMax) >= 2 {
		maxV = minMax[1]
	}
	if minV > maxV {
		minV, maxV = maxV, minV
	}
	// Inclusive range.
	span := maxV - minV + 1
	if span <= 0 {
		return absInt64(minV)
	}
	n, err := rand.Int(rand.Reader, big.NewInt(span))
	if err != nil {
		return absInt64(minV)
	}
	return absInt64(minV + n.Int64())
}

// AbsInt is WP absint — absolute value of an integer (P2.103 helper).
func AbsInt(n int64) int64 {
	return absInt64(n)
}

func absInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
