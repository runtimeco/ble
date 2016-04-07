package hci

import (
	"fmt"
	"log"
	"sync"

	"github.com/currantlabs/bt/hci/evt"
)

func newEvtHandler() *evtHandler {
	todo := func(b []byte) {
		log.Printf("hci: unhandled (TODO) event packet: [ % X ]", b)
	}

	e := &evtHandler{
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
	e.SetEventHandler(0x3E, HandlerFunc(e.handleLEMeta))
	e.SetSubeventHandler(evt.LEAdvertisingReportEvent{}.SubCode(), HandlerFunc(defaultLEAdvertisingReportHandler))
	return e
}

type evtHandler struct {
	sync.Mutex
	evth map[int]Handler
	subh map[int]Handler
}

func (e *evtHandler) EventHandler(c int) Handler {
	e.Lock()
	defer e.Unlock()
	return e.evth[c]
}

func (e *evtHandler) SetEventHandler(c int, f Handler) Handler {
	e.Lock()
	defer e.Unlock()
	old := e.evth[c]
	e.evth[c] = f
	return old
}

func (e *evtHandler) SubeventHandler(c int) Handler {
	e.Lock()
	defer e.Unlock()
	return e.subh[c]
}

func (e *evtHandler) SetSubeventHandler(c int, f Handler) Handler {
	e.Lock()
	defer e.Unlock()
	old := e.subh[c]
	e.subh[c] = f
	return old
}

func (e *evtHandler) handle(b []byte) {
	e.Lock()
	defer e.Unlock()
	code, plen := int(b[0]), int(b[1])
	if plen != len(b[2:]) {
		log.Printf("hci: corrupt event packet: [ % X ]", b)
	}
	if f, found := e.evth[code]; found {
		go f.Handle(b[2:])
		return
	}
	log.Printf("hci: unsupported event packet: [ % X ]", b)
}

func (e *evtHandler) handleLEMeta(b []byte) {
	code := int(b[0])
	if f := e.SubeventHandler(code); f != nil {
		f.Handle(b)
		return
	}
	log.Printf("Unsupported LE event: [ % X ]", b)
}

func defaultLEAdvertisingReportHandler(p []byte) {
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
