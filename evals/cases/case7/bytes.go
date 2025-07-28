// inlined from:
//
//	https://raw.githubusercontent.com/buger/jsonparser/61b32cfdfa0f5d368ef7c7daef28ce12d538740f/bytes_safe.go
//
// and
//
//	https://github.com/buger/jsonparser/blob/61b32cfdfa0f5d368ef7c7daef28ce12d538740f/bytes.go
package jsonparser

import (
	"strconv"
)

const absMinInt64 = 1 << 63
const maxInt64 = 1<<63 - 1
const maxUint64 = 1<<64 - 1

// See fastbytes_unsafe.go for explanation on why *[]byte is used (signatures must be consistent with those in that file)

func equalStr(b *[]byte, s string) bool {
	return string(*b) == s
}

func parseFloat(b *[]byte) (float64, error) {
	return strconv.ParseFloat(string(*b), 64)
}

func bytesToString(b *[]byte) string {
	return string(*b)
}

func StringToBytes(s string) []byte {
	return []byte(s)
}

// About 2x faster then strconv.ParseInt because it only supports base 10, which is enough for JSON
func parseInt(bytes []byte) (v int64, ok bool, overflow bool) {
	if len(bytes) == 0 {
		return 0, false, false
	}

	var neg bool = false
	if bytes[0] == '-' {
		neg = true
		bytes = bytes[1:]
	}

	var n uint64 = 0
	for _, c := range bytes {
		if c < '0' || c > '9' {
			return 0, false, false
		}
		if n > maxUint64/10 {
			return 0, false, true
		}
		n *= 10
		n1 := n + uint64(c-'0')
		if n1 < n {
			return 0, false, true
		}
		n = n1
	}

	if n > maxInt64 {
		if neg && n == absMinInt64 {
			return -absMinInt64, true, false
		}
		return 0, false, true
	}

	if neg {
		return -int64(n), true, false
	} else {
		return int64(n), true, false
	}
}
