package att

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// A UUID is a BLE UUID.
type UUID []byte

// UUID16 converts a uint16 (such as 0x1800) to a UUID.
func UUID16(i uint16) UUID {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, i)
	return UUID(b)
}

// Len returns the length of the UUID, in bytes.
// BLE UUIDs are either 2 or 16 bytes.
func (u UUID) Len() int { return len(u) }

// String hex-encodes a UUID.
func (u UUID) String() string { return fmt.Sprintf("%X", reverse(u)) }

// Equal returns a boolean reporting whether v represent the same UUID as u.
func (u UUID) Equal(v UUID) bool { return bytes.Equal(u, v) }

// reverse returns a reversed copy of u.
func reverse(u []byte) []byte {
	// Special-case 16 bit UUIDS for speed.
	l := len(u)
	if l == 2 {
		return []byte{u[1], u[0]}
	}
	b := make([]byte, l)
	for i := 0; i < l/2+1; i++ {
		b[i], b[l-i-1] = u[l-i-1], u[i]
	}
	return b
}
