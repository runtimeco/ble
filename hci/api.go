package hci

import (
	"io"
	"net"
)

// HCI ...
type HCI interface {
	Sender
	Dispatcher
	DataPacketHandler

	// LocalAddr returns the MAC address of local skt.
	LocalAddr() net.HardwareAddr

	// Close stop the skt.
	Close() error
}

// Command ...
type Command interface {
	OpCode() int
	Len() int
	Marshal([]byte) error
}

// CommandRP ...
type CommandRP interface {
	Unmarshal(b []byte) error
}

// Sender ...
type Sender interface {
	// Send sends a HCI Command and returns unserialized return parameter.
	Send(Command, CommandRP) error
}

// A Handler handles an HCI event packets.
type Handler interface {
	Handle([]byte)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as packet or event handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler object that calls f.
type HandlerFunc func(b []byte)

// Handle handles an event packet.
func (f HandlerFunc) Handle(b []byte) {
	f(b)
}

// Dispatcher ...
type Dispatcher interface {
	// SetEventHandler registers the handler to handle the HCI event, and returns current handler.
	SetEventHandler(c int, h Handler) Handler

	// SetSubeventHandler registers the handler to handle the HCI subevent, and returns current handler.
	SetSubeventHandler(c int, h Handler) Handler
}

// DataPacketHandler ...
type DataPacketHandler interface {
	SetDataPacketHandler(func([]byte)) (w io.Writer, size int, cnt int)
}
