package hci

import (
	"io"
	"net"

	log "github.com/Sirupsen/logrus"
	"github.com/currantlabs/bt/hci/acl"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/hci/device"
	"github.com/currantlabs/bt/hci/evt"
)

// HCI ...
type HCI interface {
	cmd.Sender
	evt.Dispatcher
	acl.DataPacketHandler

	// LocalAddr returns the MAC address of local device.
	LocalAddr() net.HardwareAddr

	// Close stop the device.
	Close() error
}

type hci struct {
	dev io.ReadWriteCloser

	sentCmds map[int]*cmdPkt
	chCmdPkt chan *cmdPkt

	// Host to Controller command flow control [Vol 2, Part E, 4.4]
	chCmdBufs chan []byte

	// HCI event handling.
	evtHanlders    *dispatcher
	subevtHandlers *dispatcher

	// Device information or status.
	addr    net.HardwareAddr
	txPwrLv int

	// ACL data packet handling.
	bufSize       int
	bufCnt        int
	handleACLData func([]byte)
}

// NewHCI ...
func NewHCI(devID int, chk bool) (HCI, error) {
	dev, err := device.NewDevice(devID, chk)
	if err != nil {
		return nil, err
	}

	h := &hci{
		dev: dev,

		chCmdPkt:  make(chan *cmdPkt),
		chCmdBufs: make(chan []byte, 8),
		sentCmds:  make(map[int]*cmdPkt),
	}

	todo := func(b []byte) {
		log.Errorf("hci: unhandled (TODO) event packet: [ % X ]", b)
	}

	h.evtHanlders = &dispatcher{
		handlers: map[int]evt.Handler{
			evt.EncryptionChangeEvent{}.Code():                     evt.HandlerFunc(todo),
			evt.ReadRemoteVersionInformationCompleteEvent{}.Code(): evt.HandlerFunc(todo),
			evt.CommandCompleteEvent{}.Code():                      evt.HandlerFunc(h.handleCommandComplete),
			evt.CommandStatusEvent{}.Code():                        evt.HandlerFunc(h.handleCommandStatus),
			evt.HardwareErrorEvent{}.Code():                        evt.HandlerFunc(todo),
			evt.DataBufferOverflowEvent{}.Code():                   evt.HandlerFunc(todo),
			evt.EncryptionKeyRefreshCompleteEvent{}.Code():         evt.HandlerFunc(todo),
			0x3E: evt.HandlerFunc(h.handleLEMeta), // FIMXE: ugliness
			evt.AuthenticatedPayloadTimeoutExpiredEvent{}.Code(): evt.HandlerFunc(todo),
		},
	}

	h.subevtHandlers = &dispatcher{
		handlers: map[int]evt.Handler{
			evt.LEAdvertisingReportEvent{}.SubCode():                evt.HandlerFunc(h.handleLEAdvertisingReport),
			evt.LEReadRemoteUsedFeaturesCompleteEvent{}.SubCode():   evt.HandlerFunc(todo),
			evt.LERemoteConnectionParameterRequestEvent{}.SubCode(): evt.HandlerFunc(todo),
		},
	}

	go h.mainLoop()
	go h.cmdLoop()
	h.chCmdBufs <- make([]byte, 64)

	return h, h.init()
}

// SetEventHandler registers the handler to handle the hci event, and returns current handler.
func (h *hci) SetEventHandler(c int, f evt.Handler) evt.Handler {
	return h.evtHanlders.SetHandler(c, f)
}

// SetSubeventHandler registers the handler to handle the hci subevent, and returns current handler.
func (h *hci) SetSubeventHandler(c int, f evt.Handler) evt.Handler {
	return h.subevtHandlers.SetHandler(c, f)
}

// LocalAddr ...
func (h *hci) LocalAddr() net.HardwareAddr {
	return h.addr
}

// Close ...
func (h *hci) Close() error {
	return h.dev.Close()
}

// Send sends a hci Command and returns unserialized return parameter.
func (h *hci) Send(c cmd.Command, r cmd.CommandRP) error {
	p := &cmdPkt{c, make(chan []byte)}
	h.chCmdPkt <- p
	b := <-p.done
	if r == nil {
		return nil
	}
	return r.Unmarshal(b)
}

// BufferInfo ...
func (h *hci) BufferInfo() (size int, cnt int) {
	return h.bufSize, h.bufCnt
}

// SetDataPacketHandler
func (h *hci) SetDataPacketHandler(f func([]byte)) {
	h.handleACLData = f
}

// Write ...
func (h *hci) Write(p []byte) (int, error) {
	return h.dev.Write(p)
}

// Read ...
func (h *hci) Read(p []byte) (int, error) {
	return h.dev.Read(p)
}

type cmdPkt struct {
	cmd  cmd.Command
	done chan []byte
}

func (h *hci) cmdLoop() {
	for p := range h.chCmdPkt {
		b := <-h.chCmdBufs
		c := p.cmd
		b[0] = byte(pktTypeCommand) // HCI header
		b[1] = byte(c.OpCode())
		b[2] = byte(c.OpCode() >> 8)
		b[3] = byte(c.Len())
		if err := c.Marshal(b[4:]); err != nil {
			log.Errorf("hci: failed to marshal cmd")
			return
		}

		h.sentCmds[c.OpCode()] = p // TODO: lock
		if n, err := h.dev.Write(b[:4+c.Len()]); err != nil {
			log.Errorf("hci: failed to send cmd")
		} else if n != 4+c.Len() {
			log.Errorf("hci: failed to send whole cmd pkt to hci socket")
		}
	}
}

func (h *hci) mainLoop() {
	b := make([]byte, 4096)
	for {
		n, err := h.dev.Read(b)
		if err != nil {
			return
		}
		if n == 0 {
			return
		}
		p := make([]byte, n)
		copy(p, b)
		h.handlePkt(p)
	}
}

func (h *hci) handlePkt(b []byte) {
	// Strip the HCI header, and pass down the rest of the packet.
	t, b := b[0], b[1:]
	switch t {
	case pktTypeCommand:
		log.Errorf("hci: unmanaged cmd: [ % X ]", b)
	case pktTypeACLData:
		h.handleACLData(b)
	case pktTypeSCOData:
		log.Errorf("hci: unsupported sco packet: [ % X ]", b)
	case pktTypeEvent:
		go h.evtHanlders.dispatch(b)
	case pktTypeVendor:
		log.Errorf("hci: unsupported vendor packet: [ % X ]", b)
	default:
		log.Errorf("hci: invalid packet: 0x%02X [ % X ]", t, b)
	}
}

func (h *hci) init() error {
	ResetRP := cmd.ResetRP{}
	if err := h.Send(&cmd.Reset{}, &ResetRP); err != nil {
		return err
	}

	ReadBDADDRRP := cmd.ReadBDADDRRP{}
	if err := h.Send(&cmd.ReadBDADDR{}, &ReadBDADDRRP); err != nil {
		return err
	}
	a := ReadBDADDRRP.BDADDR
	h.addr = net.HardwareAddr([]byte{a[5], a[4], a[3], a[2], a[1], a[0]})

	ReadLocalSupportedCommandsRP := cmd.ReadLocalSupportedCommandsRP{}
	if err := h.Send(&cmd.ReadLocalSupportedCommands{}, &ReadLocalSupportedCommandsRP); err != nil {
		return err
	}

	ReadLocalSupportedFeaturesRP := cmd.ReadLocalSupportedFeaturesRP{}
	if err := h.Send(&cmd.ReadLocalSupportedFeatures{}, &ReadLocalSupportedFeaturesRP); err != nil {
		return err
	}

	ReadLocalVersionInformationRP := cmd.ReadLocalVersionInformationRP{}
	if err := h.Send(&cmd.ReadLocalVersionInformation{}, &ReadLocalVersionInformationRP); err != nil {
		return err
	}

	ReadBufferSizeRP := cmd.ReadBufferSizeRP{}
	if err := h.Send(&cmd.ReadBufferSize{}, &ReadBufferSizeRP); err != nil {
		return err
	}

	// Assume the buffers are shared between ACL-U and LE-U.
	h.bufCnt = int(ReadBufferSizeRP.HCTotalNumACLDataPackets)
	h.bufSize = int(ReadBufferSizeRP.HCACLDataPacketLength)

	LEReadBufferSizeRP := cmd.LEReadBufferSizeRP{}
	if err := h.Send(&cmd.LEReadBufferSize{}, &LEReadBufferSizeRP); err != nil {
		return err
	}

	if LEReadBufferSizeRP.HCTotalNumLEDataPackets != 0 {
		// Okay, LE-U do have their own buffers.
		h.bufCnt = int(LEReadBufferSizeRP.HCTotalNumLEDataPackets)
		h.bufSize = int(LEReadBufferSizeRP.HCLEDataPacketLength)
	}

	LEReadLocalSupportedFeaturesRP := cmd.LEReadLocalSupportedFeaturesRP{}
	if err := h.Send(&cmd.LEReadLocalSupportedFeatures{}, &LEReadLocalSupportedFeaturesRP); err != nil {
		return err
	}

	LEReadSupportedStatesRP := cmd.LEReadSupportedStatesRP{}
	if err := h.Send(&cmd.LEReadSupportedStates{}, &LEReadSupportedStatesRP); err != nil {
		return err
	}

	LEReadAdvertisingChannelTxPowerRP := cmd.LEReadAdvertisingChannelTxPowerRP{}
	if err := h.Send(&cmd.LEReadAdvertisingChannelTxPower{}, &LEReadAdvertisingChannelTxPowerRP); err != nil {
		return err
	}
	h.txPwrLv = int(LEReadAdvertisingChannelTxPowerRP.TransmitPowerLevel)

	LESetEventMaskRP := cmd.LESetEventMaskRP{}
	if err := h.Send(&cmd.LESetEventMask{LEEventMask: 0x000000000000001F}, &LESetEventMaskRP); err != nil {
		return err
	}

	SetEventMaskRP := cmd.SetEventMaskRP{}
	if err := h.Send(&cmd.SetEventMask{EventMask: 0x3dbff807fffbffff}, &SetEventMaskRP); err != nil {
		return err
	}

	WriteLEHostSupportRP := cmd.WriteLEHostSupportRP{}
	if err := h.Send(&cmd.WriteLEHostSupport{LESupportedHost: 1, SimultaneousLEHost: 0}, &WriteLEHostSupportRP); err != nil {
		return err
	}

	WriteClassOfDeviceRP := cmd.WriteClassOfDeviceRP{}
	if err := h.Send(&cmd.WriteClassOfDevice{ClassOfDevice: [3]byte{0x40, 0x02, 0x04}}, &WriteClassOfDeviceRP); err != nil {
		return err
	}

	return nil
}
