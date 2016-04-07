package l2cap

import (
	"io"
	"log"
	"sync"

	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/hci/evt"
)

// LE implements L2CAP (LE-U logical link) handling
type LE struct {
	hci       hci.HCI
	pktWriter io.Writer

	// Host to Controller Data Flow Control Packet-based Data flow control for LE-U [Vol 2, Part E, 4.1.1]
	// Minimum 27 bytes. 4 bytes of L2CAP Header, and 23 bytes Payload from upper layer (ATT)
	pool *Pool

	// L2CAP connections
	muConns *sync.Mutex
	conns   map[uint16]*conn
	chConn  chan *conn
}

// NewL2CAP ...
func NewL2CAP(h hci.HCI) *LE {
	l := &LE{
		hci: h,

		muConns: &sync.Mutex{},
		conns:   make(map[uint16]*conn),
		chConn:  make(chan *conn),
	}

	// Pre-allocate buffers with additional head room for lower layer headers.
	// HCI header (1 Byte) + ACL Data Header (4 bytes) + L2CAP PDU (or fragment)
	w, size, cnt := h.SetDataPacketHandler(l.handleDataPacket)
	l.pktWriter = w
	l.pool = NewPool(1+4+size, cnt)

	h.SetEventHandler(evt.DisconnectionCompleteEvent{}.Code(), evt.HandlerFunc(l.handleDisconnectionComplete))
	h.SetEventHandler(evt.NumberOfCompletedPacketsEvent{}.Code(), evt.HandlerFunc(l.handleNumberOfCompletedPackets))

	h.SetSubeventHandler(evt.LEConnectionCompleteEvent{}.SubCode(), evt.HandlerFunc(l.handleLEConnectionComplete))
	h.SetSubeventHandler(evt.LEConnectionUpdateCompleteEvent{}.SubCode(), evt.HandlerFunc(l.handleLEConnectionUpdateComplete))
	h.SetSubeventHandler(evt.LELongTermKeyRequestEvent{}.SubCode(), evt.HandlerFunc(l.handleLELongTermKeyRequest))

	return l
}

func (l *LE) handleDataPacket(b []byte) {
	l.muConns.Lock()
	c, ok := l.conns[pkt(b).handle()]
	l.muConns.Unlock()
	if !ok {
		return
	}
	c.chInPkt <- b
}

// Accept returns a L2CAP connections.
func (l *LE) Accept() (Conn, error) {
	return <-l.chConn, nil
}

func (l *LE) handleLEConnectionComplete(b []byte) {
	e := &evt.LEConnectionCompleteEvent{}
	if err := e.Unmarshal(b); err != nil {
		return
	}

	c := newConn(l, e)
	l.muConns.Lock()
	l.conns[e.ConnectionHandle] = c
	l.muConns.Unlock()
	l.chConn <- c
}

func (l *LE) handleLEConnectionUpdateComplete(b []byte) {
	e := &evt.LEConnectionUpdateCompleteEvent{}
	if err := e.Unmarshal(b); err != nil {
		return
	}
}

func (l *LE) handleDisconnectionComplete(b []byte) {
	e := &evt.DisconnectionCompleteEvent{}
	if err := e.Unmarshal(b); err != nil {
		return
	}
	l.muConns.Lock()
	c, found := l.conns[e.ConnectionHandle]
	delete(l.conns, e.ConnectionHandle)
	l.muConns.Unlock()
	if !found {
		log.Printf("conns: disconnecting an invalid handle %04X", e.ConnectionHandle)
		return
	}
	close(c.chInPkt)
	c.txBuffer.FreeAll()
}

func (l *LE) handleNumberOfCompletedPackets(b []byte) {
	e := &evt.NumberOfCompletedPacketsEvent{}
	if err := e.Unmarshal(b); err != nil {
		return
	}

	l.muConns.Lock()
	defer l.muConns.Unlock()
	for i := 0; i < int(e.NumberOfHandles); i++ {
		c, ok := l.conns[e.ConnectionHandle[i]]
		if !ok {
			return
		}

		// Add the HCI buffer to the per-connection list. When written buffers are acked by
		// the controller via NumberOfCompletedPackets event, we'll put them back to the pool.
		// When a connection disconnects, all the sent packets and weren't acked yet
		// will be recylecd. [Vol2, Part E 4.1.1]
		for j := 0; j < int(e.HCNumOfCompletedPackets[i]); j++ {
			c.txBuffer.Free()
		}
	}
}

func (l *LE) handleLELongTermKeyRequest(b []byte) {
	e := &evt.LELongTermKeyRequestEvent{}
	if err := e.Unmarshal(b); err != nil {
		return
	}

	l.hci.Send(&cmd.LELongTermKeyRequestNegativeReply{
		ConnectionHandle: e.ConnectionHandle,
	}, nil)
}
