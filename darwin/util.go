package darwin

import (
	"github.com/runtimeco/ble"
)

func uuidSlice(uu []ble.UUID) [][]byte {
	us := [][]byte{}
	for _, u := range uu {
		us = append(us, ble.Reverse(u))
	}
	return us
}

func uuidStrWithDashes(s string) string {
	if len(s) != 32 {
		return s
	}

	// 01234567-89ab-cdef-0123-456789abcdef
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
