package gatt

import (
	"encoding/binary"
	"log"

	"golang.org/x/net/context"

	"github.com/currantlabs/bt/att"
)

const (
	keyData = iota
	keyOffset
	keyCentral
	keyNotifier
)

// Data ...
func Data(ctx context.Context) []byte { return ctx.Value(keyData).([]byte) }

// WithData ...
func WithData(ctx context.Context, d []byte) context.Context {
	return context.WithValue(ctx, keyData, d)
}

// Offset ...
func Offset(ctx context.Context) int { return ctx.Value(keyOffset).(int) }

// WithOffset ...
func WithOffset(ctx context.Context, o int) context.Context {
	return context.WithValue(ctx, keyOffset, o)
}

// central ...
func central(ctx context.Context) *Central { return ctx.Value(keyCentral).(*Central) }

// withCentral ...
func withCentral(ctx context.Context, c *Central) context.Context {
	return context.WithValue(ctx, keyCentral, c)
}

// Notifier ...
func Notifier(ctx context.Context) *notifier { return ctx.Value(keyNotifier).(*notifier) }

// WithNotifier ...
func WithNotifier(ctx context.Context, n *notifier) context.Context {
	return context.WithValue(ctx, keyNotifier, n)
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

func setupCCCD(c *Characteristic, h Handler) *Descriptor {
	d := c.cccd
	if d == nil {
		d = c.AddDescriptor(attrClientCharacteristicConfigUUID)
		c.cccd = d
	}

	var ccc uint16
	n := &notifier{char: c}
	i := &notifier{char: c}

	d.Handle(
		CharRead,
		HandlerFunc(func(ctx context.Context, resp *ResponseWriter) {
			binary.Write(resp, binary.LittleEndian, ccc)
		}))

	d.Handle(
		CharWrite|CharWriteNR,
		HandlerFunc(func(ctx context.Context, resp *ResponseWriter) {
			data := Data(ctx)
			central := central(ctx)
			if len(data) != 2 {
				resp.SetStatus(att.ErrInvalAttrValueLen)
				return
			}
			new := binary.LittleEndian.Uint16(data)
			log.Printf("CCC: 0x%04X -> 0x%04X", ccc, new)

			i.send = central.server.Indicate
			n.send = central.server.Notify

			i.done = new&flagCCCIndicate == 0
			n.done = new&flagCCCNotify == 0

			if !i.done && ccc&flagCCCIndicate == 0 {
				go h.Serve(WithNotifier(ctx, i), resp)
			}
			if !n.done && ccc&flagCCCNotify == 0 {
				go h.Serve(WithNotifier(ctx, n), resp)
			}
			ccc = new
		}))
	return d
}

// Handle ...
func (c *Characteristic) Handle(p Property, h Handler) *Characteristic {

	c.value[p&CharRead] = h
	c.value[p&CharWriteNR] = h
	c.value[p&CharWrite] = h
	c.value[p&CharSignedWrite] = h
	c.value[p&CharExtended] = h

	if p&(CharNotify|CharIndicate) != 0 {
		setupCCCD(c, h)
	}

	c.Property |= p
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
	gattResp := &ResponseWriter{resp: resp, status: att.ErrSuccess}
	var h Handler
	switch req[0] {
	case att.ReadByTypeRequestCode:
		if h = v[CharRead]; h == nil {
			return att.ErrReadNotPerm
		}
	case att.ReadRequestCode:
		if h = v[CharRead]; h == nil {
			return att.ErrReadNotPerm
		}
	case att.ReadBlobRequestCode:
		if h = v[CharRead]; h == nil {
			return att.ErrReadNotPerm
		}
		ctx = WithOffset(ctx, int(att.ReadBlobRequest(req).ValueOffset()))
	case att.WriteRequestCode:
		if h = v[CharWrite]; h == nil {
			return att.ErrWriteNotPerm
		}
		ctx = WithData(ctx, att.WriteRequest(req).AttributeValue())
	case att.WriteCommandCode:
		if h = v[CharWriteNR]; h == nil {
			return att.ErrWriteNotPerm
		}
		ctx = WithData(ctx, att.WriteRequest(req).AttributeValue())
	case att.PrepareWriteRequestCode:
	case att.ExecuteWriteRequestCode:
	case att.SignedWriteCommandCode:
	// case att.ReadByGroupTypeRequestCode:
	// case att.ReadMultipleRequestCode:
	default:
	}

	h.Serve(ctx, gattResp)
	return gattResp.status
}

func (v attValue) setvalue(value []byte) {
	v[CharRead] = HandlerFunc(func(ctx context.Context, resp *ResponseWriter) {
		resp.Write(value)
	})
}

// A Notifier provides a means for a GATT server to send notifications about value changes to a connected device.
type notifier struct {
	char   *Characteristic
	maxlen int
	done   bool
	send   func(uint16, []byte) (int, error)
}

// Write sends data to the central.
func (n *notifier) Write(b []byte) (int, error) {
	return n.send(n.char.vh, b)
}

// Cap returns the maximum number of bytes that may be sent in a single notification.
func (n *notifier) Cap() int { return n.maxlen }

// Done reports whether the central has requested not to receive any more notifications with this notifier.
func (n *notifier) Done() bool { return n.done }
func (n *notifier) stop()      { n.done = true }

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
	Serve(ctx context.Context, resp *ResponseWriter)
}

// HandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type HandlerFunc func(ctx context.Context, resp *ResponseWriter)

// Serve returns f(r, maxlen, offset).
func (f HandlerFunc) Serve(ctx context.Context, resp *ResponseWriter) {
	f(ctx, resp)
}
