package acl

import "io"

// DataPacketHandler ...
type DataPacketHandler interface {
	io.ReadWriter

	BufferInfo() (size int, cnt int)
	SetDataPacketHandler(func([]byte))
}
