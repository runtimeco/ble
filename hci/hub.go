package hci

import (
	"fmt"
	"log"
	"sync"

	"github.com/currantlabs/bt/hci/evt"
)

func newEvtHub() *evtHub {
	todo := func(b []byte) error {
		return fmt.Errorf("hci: unhandled (TODO) event packet: [ % X ]", b)
	}

	h := &evtHub{
		evth: map[int]Handler{
			evt.EncryptionChangeEvent{}.Code():                     HandlerFunc(todo),
			evt.ReadRemoteVersionInformationCompleteEvent{}.Code(): HandlerFunc(todo),
			evt.HardwareErrorEvent{}.Code():                        HandlerFunc(todo),
			evt.DataBufferOverflowEvent{}.Code():                   HandlerFunc(todo),
			evt.EncryptionKeyRefreshCompleteEvent{}.Code():         HandlerFunc(todo),
			evt.AuthenticatedPayloadTimeoutExpiredEvent{}.Code():   HandlerFunc(todo),
		},
		subh: map[int]Handler{
			evt.LEReadRemoteUsedFeaturesCompleteEvent{}.SubCode():   HandlerFunc(todo),
			evt.LERemoteConnectionParameterRequestEvent{}.SubCode(): HandlerFunc(todo),
		},
	}
	h.SetEventHandler(0x3E, HandlerFunc(h.handleLEMeta))
	h.SetSubeventHandler(evt.LEAdvertisingReportEvent{}.SubCode(), HandlerFunc(h.handleLEAdvertisingReport))
	return h
}

type evtHub struct {
	sync.Mutex
	evth map[int]Handler
	subh map[int]Handler
}

func (h *evtHub) EventHandler(c int) Handler {
	h.Lock()
	defer h.Unlock()
	return h.evth[c]
}

func (h *evtHub) SetEventHandler(c int, f Handler) Handler {
	h.Lock()
	defer h.Unlock()
	old := h.evth[c]
	h.evth[c] = f
	return old
}

func (h *evtHub) SubeventHandler(c int) Handler {
	h.Lock()
	defer h.Unlock()
	return h.subh[c]
}

func (h *evtHub) SetSubeventHandler(c int, f Handler) Handler {
	h.Lock()
	defer h.Unlock()
	old := h.subh[c]
	h.subh[c] = f
	return old
}

func (h *evtHub) handle(b []byte) error {
	h.Lock()
	defer h.Unlock()
	code, plen := int(b[0]), int(b[1])
	if plen != len(b[2:]) {
		return fmt.Errorf("hci: corrupt event packet: [ % X ]", b)
	}
	if f, found := h.evth[code]; found {
		return f.Handle(b[2:])
	}
	return fmt.Errorf("hci: unsupported event packet: [ % X ]", b)
}

func (h *evtHub) handleLEMeta(b []byte) error {
	code := int(b[0])
	if f := h.SubeventHandler(code); f != nil {
		return f.Handle(b)
	}
	return fmt.Errorf("hci: unsupported LE event: [ % X ]", b)
}

func (h *evtHub) handleLEAdvertisingReport(p []byte) error {
	e := &evt.LEAdvertisingReportEvent{}
	if err := e.Unmarshal(p); err != nil {
		return err
	}
	f := func(a [6]byte) string {
		return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", a[5], a[4], a[3], a[2], a[1], a[0])
	}
	for i := 0; i < int(e.NumReports); i++ {
		log.Printf("%d, %d, %s, %d, [% X]",
			e.EventType[i], e.AddressType[i], f(e.Address[i]), e.RSSI[i], e.Data[i])
	}
	return nil
}
