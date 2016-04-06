package hci

import (
	"fmt"
	"log"
	"sync"

	"github.com/currantlabs/bt/hci/evt"
)

type dispatcher struct {
	sync.Mutex
	handlers map[int]evt.Handler
}

func (d *dispatcher) Handler(c int) evt.Handler {
	d.Lock()
	defer d.Unlock()
	return d.handlers[c]
}

func (d *dispatcher) SetHandler(c int, f evt.Handler) evt.Handler {
	d.Lock()
	defer d.Unlock()
	old := d.handlers[c]
	d.handlers[c] = f
	return old
}

func (d *dispatcher) dispatch(b []byte) {
	d.Lock()
	defer d.Unlock()
	code, plen := int(b[0]), int(b[1])
	if plen != len(b[2:]) {
		log.Printf("hci: corrupt event packet: [ % X ]", b)
	}
	if f, found := d.handlers[code]; found {
		go f.Handle(b[2:])
		return
	}
	log.Printf("hci: unsupported event packet: [ % X ]", b)
}

func (h *hci) handleCommandComplete(b []byte) {
	var e evt.CommandCompleteEvent
	if err := e.Unmarshal(b); err != nil {
		return
	}
	for i := 0; i < int(e.NumHCICommandPackets); i++ {
		h.chCmdBufs <- make([]byte, 64)
	}
	if e.CommandOpcode == 0x0000 {
		// NOP command, used for flow control purpose [Vol 2, Part E, 4.4]
		return
	}
	p, found := h.sentCmds[int(e.CommandOpcode)]
	if !found {
		log.Printf("event: can't find the cmd for CommandCompleteEP: %v", e)
		return
	}
	p.done <- e.ReturnParameters
}

func (h *hci) handleCommandStatus(b []byte) {
	var e evt.CommandStatusEvent
	if err := e.Unmarshal(b); err != nil {
		return
	}
	for i := 0; i < int(e.NumHCICommandPackets); i++ {
		h.chCmdBufs <- make([]byte, 64)
	}
	p, found := h.sentCmds[int(e.CommandOpcode)]
	if !found {
		log.Printf("event: can't find the cmd for CommandStatusEP: %v", e)
		return
	}
	close(p.done)
}

func (h *hci) handleLEMeta(b []byte) {
	code := int(b[0])
	if f := h.subevtHandlers.Handler(code); f != nil {
		f.Handle(b)
		return
	}
	log.Printf("Unsupported LE event: [ % X ]", b)
}

func (h *hci) handleLEAdvertisingReport(p []byte) {
	e := &evt.LEAdvertisingReportEvent{}
	if err := e.Unmarshal(p); err != nil {
		return
	}
	f := func(a [6]byte) string {
		return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", a[5], a[4], a[3], a[2], a[1], a[0])
	}
	for i := 0; i < int(e.NumReports); i++ {
		log.Printf("%d, %d, %s, %d, [% X]",
			e.EventType[i], e.AddressType[i], f(e.Address[i]), e.RSSI[i], e.Data[i])
	}
}
