package gatt

import (
	"encoding/binary"
	"log"

	"golang.org/x/net/context"

	"github.com/currantlabs/bt/att"
)

// A Request is the context for a request from a connected central device.
type Request struct {
	Central  *Central
	Cap      int // maximum allowed reply length
	Offset   int // request value offset
	Data     []byte
	Notifier *Notifier
}

// Property ...
type Property int

// Characteristic property flags (spec 3.3.3.1)
const (
	CharBroadcast   Property = 0x01 // may be brocasted
	CharRead        Property = 0x02 // may be read
	CharWriteNR     Property = 0x04 // may be written to, with no reply
	CharWrite       Property = 0x08 // may be written to, with a reply
	CharNotify      Property = 0x10 // supports notifications
	CharIndicate    Property = 0x20 // supports Indications
	CharSignedWrite Property = 0x40 // supports signed write
	CharExtended    Property = 0x80 // supports extended properties
)

// A Service is a BLE service.
type Service struct {
	UUID            UUID
	Characteristics []*Characteristic

	h    uint16
	endh uint16
}

// NewService ...
func NewService(u UUID) *Service {
	return &Service{UUID: u}
}

// AddCharacteristic adds a characteristic to a service.
func (s *Service) AddCharacteristic(u UUID) *Characteristic {
	c := &Characteristic{UUID: u, value: make(map[Property]Handler)}
	s.Characteristics = append(s.Characteristics, c)
	return c
}

// Name returns the specificatin name of the service according to its UUID.
func (s *Service) Name() string { return knownServices[s.UUID.String()].Name }

// A Characteristic is a BLE characteristic.
type Characteristic struct {
	UUID        UUID
	Property    Property // enabled properties
	Descriptors []*Descriptor

	cccd *Descriptor

	value attValue

	h    uint16
	vh   uint16
	endh uint16
}

// Name returns the specificatin name of the characteristic.
func (c *Characteristic) Name() string {
	return knownCharacteristics[c.UUID.String()].Name
}

// Handle ...
func (c *Characteristic) Handle(p Property, h Handler) *Characteristic {
	c.Property |= p

	c.value[p&CharRead] = h
	c.value[p&CharWriteNR] = h
	c.value[p&CharWrite] = h
	c.value[p&CharSignedWrite] = h
	c.value[p&CharExtended] = h

	if p&(CharNotify|CharIndicate) == 0 {
		return c
	}

	d := c.cccd
	if d == nil {
		d = c.AddDescriptor(attrClientCharacteristicConfigUUID)
		c.cccd = d
	}

	var ccc uint16
	n := &Notifier{char: c}
	i := &Notifier{char: c}

	d.value[CharRead] = HandlerFunc(func(resp *ResponseWriter, req *Request) {
		log.Printf("CCCD: read: 0x%04X", ccc)
		binary.Write(resp, binary.LittleEndian, ccc)
	})

	d.value[CharWrite] = HandlerFunc(func(resp *ResponseWriter, req *Request) {
		if len(req.Data) != 2 {
			resp.SetStatus(att.ErrInvalAttrValueLen)
			return
		}
		v := binary.LittleEndian.Uint16(req.Data)
		log.Printf("CCC: 0x%04X -> 0x%04X", ccc, v)
		if (ccc&gattCCCIndicateFlag) == 0 && v&gattCCCIndicateFlag != 0 {
			req.Notifier = i
			i.send = req.Central.server.SendIndication
			i.done = false
			go h.Serve(resp, req)
		}
		if (ccc&gattCCCIndicateFlag) != 0 && v&gattCCCIndicateFlag == 0 {
			i.done = true
		}
		if (ccc&gattCCCNotifyFlag) == 0 && v&gattCCCNotifyFlag != 0 {
			req.Notifier = n
			n.send = req.Central.server.SendNotification
			n.done = false
			go h.Serve(resp, req)
		}
		if (ccc&gattCCCNotifyFlag) != 0 && v&gattCCCNotifyFlag == 0 {
			n.done = true
		}
		ccc = v
	})
	return c
}

// SetValue ...
func (c *Characteristic) SetValue(value []byte) {
	c.value.setvalue(value)
}

// AddDescriptor adds a descriptor to a characteristic.
func (c *Characteristic) AddDescriptor(u UUID) *Descriptor {
	d := &Descriptor{UUID: u, value: make(map[Property]Handler)}
	c.Descriptors = append(c.Descriptors, d)
	return d
}

// Descriptor is a BLE descriptor
type Descriptor struct {
	UUID     UUID
	Property Property // enabled properties

	h uint16

	value attValue
}

// Name returns the specificatin name of the descriptor.
func (d *Descriptor) Name() string {
	return knownDescriptors[d.UUID.String()].Name
}

// Handle ...
func (d *Descriptor) Handle(p Property, h Handler) *Descriptor {
	if p&^(CharRead|CharWrite|CharWriteNR) != 0 {
		panic("Invalid Property")
	}
	d.value[p&CharRead] = h
	d.value[p&CharWrite] = h
	d.value[p&CharWriteNR] = h
	d.Property |= p
	return d
}

// SetValue ...
func (d *Descriptor) SetValue(value []byte) {
	d.value.setvalue(value)
}

type attValue map[Property]Handler

// Handle ...
func (v attValue) Handle(ctx context.Context, req []byte, resp *att.ResponseWriter) att.Error {
	central := ctx.Value("central").(*Central)
	gattResp := &ResponseWriter{resp: resp, status: att.ErrSuccess}
	switch req[0] {
	case att.ReadRequestCode:
		h := v[CharRead]
		if h == nil {
			return att.ErrReadNotPerm
		}
		r := &Request{Central: central}
		h.Serve(gattResp, r)
	case att.ReadBlobRequestCode:
		h := v[CharRead]
		if h == nil {
			return att.ErrReadNotPerm
		}
		offset := att.ReadBlobRequest(req).ValueOffset()
		r := &Request{Central: central, Offset: int(offset)}
		h.Serve(gattResp, r)
	case att.WriteRequestCode:
		h := v[CharWrite]
		if h == nil {
			return att.ErrWriteNotPerm
		}
		data := att.WriteRequest(req).AttributeValue()
		r := &Request{Central: central, Data: data, Offset: 0}
		h.Serve(gattResp, r)
	case att.WriteCommandCode:
		h := v[CharWriteNR]
		if h == nil {
			return att.ErrWriteNotPerm
		}
		data := att.WriteRequest(req).AttributeValue()
		r := &Request{Central: central, Data: data, Offset: 0}
		h.Serve(gattResp, r)
	case att.PrepareWriteRequestCode:
	case att.ExecuteWriteRequestCode:
	case att.SignedWriteCommandCode:
	// case att.ReadByGroupTypeRequestCode:
	// case att.ReadByTypeRequestCode:
	// case att.ReadMultipleRequestCode:
	default:
	}

	return gattResp.status
}

func (v attValue) setvalue(value []byte) {
	v[CharRead] = HandlerFunc(func(resp *ResponseWriter, req *Request) {
		resp.Write(value)
	})
}

// A Notifier provides a means for a GATT server to send notifications about value changes to a connected device.
type Notifier struct {
	char   *Characteristic
	maxlen int
	done   bool
	send   func(uint16, []byte) (int, error)
}

// Write sends data to the central.
func (n *Notifier) Write(b []byte) (int, error) {
	return n.send(n.char.vh, b)
}

// Cap returns the maximum number of bytes that may be sent in a single notification.
func (n *Notifier) Cap() int { return n.maxlen }

// Done reports whether the central has requested not to receive any more notifications with this notifier.
func (n *Notifier) Done() bool { return n.done }
func (n *Notifier) stop()      { n.done = true }

// ResponseWriter ...
type ResponseWriter struct {
	resp   *att.ResponseWriter
	status att.Error
}

// Write writes data to return as the characteristic value.
func (r *ResponseWriter) Write(b []byte) (int, error) { return r.resp.Write(b) }

// SetStatus reports the result of the request.
func (r *ResponseWriter) SetStatus(status att.Error) { r.status = status }

// A Handler handles GATT requests.
type Handler interface {
	Serve(resp *ResponseWriter, req *Request)
}

// HandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type HandlerFunc func(resp *ResponseWriter, req *Request)

// Serve returns f(r, maxlen, offset).
func (f HandlerFunc) Serve(resp *ResponseWriter, req *Request) {
	f(resp, req)
}
