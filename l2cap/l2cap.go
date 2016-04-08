package l2cap

import (
	"fmt"
	"io"
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
	w, size, cnt := h.SetACLProcessor(l.handlePacket)
	l.pktWriter = w
	l.pool = NewPool(1+4+size, cnt)

	h.SetEventHandler(evt.DisconnectionCompleteEvent{}.Code(), hci.HandlerFunc(l.handleDisconnectionComplete))
	h.SetEventHandler(evt.NumberOfCompletedPacketsEvent{}.Code(), hci.HandlerFunc(l.handleNumberOfCompletedPackets))

	h.SetSubeventHandler(evt.LEConnectionCompleteEvent{}.SubCode(), hci.HandlerFunc(l.handleLEConnectionComplete))
	h.SetSubeventHandler(evt.LEConnectionUpdateCompleteEvent{}.SubCode(), hci.HandlerFunc(l.handleLEConnectionUpdateComplete))
	h.SetSubeventHandler(evt.LELongTermKeyRequestEvent{}.SubCode(), hci.HandlerFunc(l.handleLELongTermKeyRequest))

	return l
}

func (l *LE) handlePacket(b []byte) error {
	l.muConns.Lock()
	c, ok := l.conns[pkt(b).handle()]
	l.muConns.Unlock()
	if !ok {
		return fmt.Errorf("l2cap: incoming packet for non-existing connection")
	}
	c.chInPkt <- b
	return nil
}

// Accept returns a L2CAP connections.
func (l *LE) Accept() (Conn, error) {
	return <-l.chConn, nil
}

func (l *LE) handleLEConnectionComplete(b []byte) error {
	e := &evt.LEConnectionCompleteEvent{}
	if err := e.Unmarshal(b); err != nil {
		return err
	}

	c := newConn(l, e)
	l.muConns.Lock()
	l.conns[e.ConnectionHandle] = c
	l.muConns.Unlock()
	l.chConn <- c
	return nil
}

func (l *LE) handleLEConnectionUpdateComplete(b []byte) error {
	e := &evt.LEConnectionUpdateCompleteEvent{}
	return e.Unmarshal(b)
}

func (l *LE) handleDisconnectionComplete(b []byte) error {
	e := &evt.DisconnectionCompleteEvent{}
	if err := e.Unmarshal(b); err != nil {
		return err
	}
	l.muConns.Lock()
	c, found := l.conns[e.ConnectionHandle]
	delete(l.conns, e.ConnectionHandle)
	l.muConns.Unlock()
	if !found {
		return fmt.Errorf("l2cap: disconnecting an invalid handle %04X", e.ConnectionHandle)
	}
	close(c.chInPkt)
	c.txBuffer.FreeAll()
	return nil
}

func (l *LE) handleNumberOfCompletedPackets(b []byte) error {
	e := &evt.NumberOfCompletedPacketsEvent{}
	if err := e.Unmarshal(b); err != nil {
		return err
	}

	l.muConns.Lock()
	defer l.muConns.Unlock()
	for i := 0; i < int(e.NumberOfHandles); i++ {
		c, found := l.conns[e.ConnectionHandle[i]]
		if !found {
			continue
		}

		// Add the HCI buffer to the per-connection list. When written buffers are acked by
		// the controller via NumberOfCompletedPackets event, we'll put them back to the pool.
		// When a connection disconnects, all the sent packets and weren't acked yet
		// will be recylecd. [Vol2, Part E 4.1.1]
		for j := 0; j < int(e.HCNumOfCompletedPackets[i]); j++ {
			c.txBuffer.Free()
		}
	}
	return nil
}

func (l *LE) handleLELongTermKeyRequest(b []byte) error {
	e := &evt.LELongTermKeyRequestEvent{}
	if err := e.Unmarshal(b); err != nil {
		return err
	}

	return l.hci.Send(&cmd.LELongTermKeyRequestNegativeReply{
		ConnectionHandle: e.ConnectionHandle,
	}, nil)
}
