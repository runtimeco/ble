package gatt

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"

	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/hci"
)

// Peripheral represent a remote peripheral device.
type Peripheral struct {
	// NameChanged is called whenever the Peripheral GAP device name has changed.
	NameChanged func(*Peripheral)

	// ServicedModified is called when one or more service of a Peripheral have changed.
	// A list of invalid service is provided in the parameter.
	ServicesModified func(*Peripheral, []*Service)

	d    *Device
	svcs []*Service

	name      string
	adv       *Advertisement
	advReport *advertisingReport
	addr      net.HardwareAddr

	handler *nHandlers
	c       *att.Client
}

// Device returns the underlying device.
func (p *Peripheral) Device() *Device {
	return p.d
}

// ID is the platform specific unique ID of the remote peripheral, e.g. MAC for Linux, Peripheral UUID for MacOS.
func (p *Peripheral) ID() string {
	return p.addr.String()
}

// Name returns the name of the remote peripheral.
// This can be the advertised name, if exists, or the GAP device name, which takes priority
func (p *Peripheral) Name() string {
	return p.name
}

// Services returnns the services of the remote peripheral which has been discovered.
func (p *Peripheral) Services() []*Service {
	return p.svcs
}

func newPeripheral(d *Device, l2c hci.Conn) *Peripheral {
	h := newNHandler()
	p := &Peripheral{
		d:       d,
		c:       att.NewClient(l2c, h),
		handler: h,
	}
	return p
}

// DiscoverServices discovers all the primary service on a server. [Vol 3, Parg G, 4.4.1]
// DiscoverServices discover the specified services of the remote peripheral.
// If the specified services is set to nil, all the available services of the remote peripheral are returned.
func (p *Peripheral) DiscoverServices(filter []UUID) ([]*Service, error) {
	start := uint16(0x0001)
	for {
		length, b, err := p.c.ReadByGroupType(start, 0xFFFF, att.UUID(attrPrimaryServiceUUID))
		if err == att.ErrAttrNotFound {
			return p.svcs, nil
		}
		if err != nil {
			return nil, err
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			endh := binary.LittleEndian.Uint16(b[2:4])
			u := UUID(b[4:length])
			if filter == nil || UUIDContains(filter, u) {
				p.svcs = append(p.svcs, &Service{UUID: u, h: h, endh: endh})
			}
			if endh == 0xFFFF {
				return p.svcs, nil
			}
			start = endh + 1
			b = b[length:]
		}
	}
}

// DiscoverIncludedServices discovers the specified included services of a service.
// If the specified services is set to nil, all the included services of the service are returned.
func (p *Peripheral) DiscoverIncludedServices(ss []UUID, s *Service) ([]*Service, error) {
	return nil, nil
}

// DiscoverCharacteristics discovers the specified characteristics of a service.
// If the specified characterstics is set to nil, all the characteristic of the service are returned.
func (p *Peripheral) DiscoverCharacteristics(filter []UUID, s *Service) ([]*Characteristic, error) {
	start := s.h
	var lastChar *Characteristic
	for start <= s.endh {
		length, b, err := p.c.ReadByType(start, s.endh, att.UUID(attrCharacteristicUUID))
		if err == att.ErrAttrNotFound {
			break
		} else if err != nil {
			return nil, err
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			props := Property(b[2])
			vh := binary.LittleEndian.Uint16(b[3:5])
			u := UUID(b[5:length])
			c := &Characteristic{UUID: u, Property: props, h: h, vh: vh, endh: s.endh}
			if filter == nil || UUIDContains(filter, u) {
				s.Characteristics = append(s.Characteristics, c)
			}
			if lastChar != nil {
				lastChar.endh = c.h - 1
			}
			lastChar = c
			start = vh + 1
			b = b[length:]
		}
	}
	return s.Characteristics, nil
}

// DiscoverDescriptors discovers the descriptors of a characteristic.
// If the specified descriptors is set to nil, all the descriptors of the characteristic are returned.
func (p *Peripheral) DiscoverDescriptors(filter []UUID, c *Characteristic) ([]*Descriptor, error) {
	start := c.vh + 1
	for start <= c.endh {
		fmt, b, err := p.c.FindInformation(start, c.endh)
		if err == att.ErrAttrNotFound {
			break
		} else if err != nil {
			return nil, err
		}
		length := 2 + 2
		if fmt == 0x02 {
			length = 2 + 16
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			u := UUID(b[2:length])
			d := &Descriptor{UUID: u, h: h}
			if filter == nil || UUIDContains(filter, u) {
				c.Descriptors = append(c.Descriptors, d)
			}
			if u.Equal(attrClientCharacteristicConfigUUID) {
				c.cccd = d
			}
			start = h + 1
			b = b[length:]
		}
	}
	return c.Descriptors, nil
}

// ReadCharacteristic retrieves the value of a specified characteristic.
func (p *Peripheral) ReadCharacteristic(c *Characteristic) ([]byte, error) { return p.c.Read(c.vh) }

// ReadLongCharacteristic retrieves the value of a specified characteristic that is longer than the MTU.
func (p *Peripheral) ReadLongCharacteristic(c *Characteristic) ([]byte, error) {
	return nil, nil
}

// WriteCharacteristic writes the value of a characteristic.
func (p *Peripheral) WriteCharacteristic(c *Characteristic, value []byte, noRsp bool) error {
	if noRsp {
		p.c.WriteCommand(c.vh, value)
		return nil
	}
	return p.c.Write(c.vh, value)
}

// ReadDescriptor retrieves the value of a specified characteristic descriptor.
func (p *Peripheral) ReadDescriptor(d *Descriptor) ([]byte, error) {
	return p.c.Read(d.h)
}

// WriteDescriptor writes the value of a characteristic descriptor.
func (p *Peripheral) WriteDescriptor(d *Descriptor, v []byte) error {
	return p.c.Write(d.h, v)
}

// ReadRSSI retrieves the current RSSI value for the remote peripheral.
func (p *Peripheral) ReadRSSI() int {
	return -1
}

// SetMTU sets the mtu for the remote peripheral.
func (p *Peripheral) SetMTU(mtu int) error {
	_, err := p.c.ExchangeMTU(mtu)
	return err
}

// NotificationHandler ...
type NotificationHandler func(req []byte)

// SetNotificationHandler sets notifications for the value of a specified characteristic.
func (p *Peripheral) SetNotificationHandler(c *Characteristic, h NotificationHandler) error {
	if c.cccd == nil {
		return errors.New("no cccd") // FIXME
	}
	return p.setHandler(c.cccd.h, c.vh, gattCCCNotifyFlag, h)
}

// SetIndicationHandler sets indications for the value of a specified characteristic.
func (p *Peripheral) SetIndicationHandler(c *Characteristic, h NotificationHandler) error {
	if c.cccd == nil {
		return errors.New("no cccd") // FIXME
	}
	return p.setHandler(c.cccd.h, c.vh, gattCCCIndicateFlag, h)
}

func (p *Peripheral) setHandler(cccdh, valueh, flag uint16, h NotificationHandler) error {
	ccc := make([]byte, 2)
	binary.LittleEndian.PutUint16(ccc, flag)
	p.handler.setHandler(cccdh, valueh, flag, h)
	if err := p.c.Write(cccdh, ccc); err != nil {
		return err
	}
	return nil
}

func (p *Peripheral) clearHandler(c *Characteristic, flag uint16) error {
	ccc := make([]byte, 2)
	if err := p.c.Write(c.cccd.h, ccc); err != nil {
		return err
	}
	p.handler.setHandler(c.cccd.h, c.vh, flag, nil)
	return nil
}

func (p *Peripheral) ClearHandlers() error {
	cccdh := make(map[uint16]uint16)
	for k, v := range p.handler.icccdhs {
		cccdh[k] = v
	}
	for k, v := range p.handler.ncccdhs {
		cccdh[k] = v
	}
	for _, cccdh := range cccdh {
		ccc := make([]byte, 2)
		if err := p.c.Write(cccdh, ccc); err != nil {
			return err
		}
	}
	return nil
}

type nHandlers struct {
	*sync.RWMutex
	iHandlers map[uint16]NotificationHandler
	nHandlers map[uint16]NotificationHandler
	icccdhs   map[uint16]uint16
	ncccdhs   map[uint16]uint16
}

func newNHandler() *nHandlers {
	h := &nHandlers{
		RWMutex:   &sync.RWMutex{},
		iHandlers: make(map[uint16]NotificationHandler),
		nHandlers: make(map[uint16]NotificationHandler),
		icccdhs:   make(map[uint16]uint16),
		ncccdhs:   make(map[uint16]uint16),
	}
	return h
}

func (n *nHandlers) HandleNotification(req []byte) {
	n.RLock()
	defer n.RUnlock()
	handlers := n.nHandlers
	if req[0] == att.HandleValueIndicationCode {
		handlers = n.iHandlers
	}
	valueh := att.HandleValueIndication(req).AttributeHandle()
	h := handlers[valueh]
	if h != nil {
		h(req[3:])
	}
}

func (n *nHandlers) setHandler(cccdh, valueh, flag uint16, h NotificationHandler) {
	n.Lock()
	defer n.Unlock()
	handlers := n.nHandlers
	cccdhs := n.ncccdhs
	if flag == gattCCCIndicateFlag {
		handlers = n.iHandlers
		cccdhs = n.icccdhs
	}
	if h == nil {
		delete(handlers, valueh)
		delete(cccdhs, valueh)
		return
	}
	handlers[valueh] = h
	cccdhs[valueh] = cccdh
}
