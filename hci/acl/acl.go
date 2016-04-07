package acl

import "io"

// DataPacketHandler ...
type DataPacketHandler interface {
	SetDataPacketHandler(func([]byte)) (w io.Writer, size int, cnt int)
}
