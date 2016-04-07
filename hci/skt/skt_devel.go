package skt

import (
	"errors"
	"log"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// IoR used for an ioctl that reads data from the skt driver.
func ioR(t, nr, size uintptr) uintptr {
	return (2 << 30) | (t << 8) | nr | (size << 16)
}

// IoW used for an ioctl that writes data to the skt driver.
func ioW(t, nr, size uintptr) uintptr {
	return (1 << 30) | (t << 8) | nr | (size << 16)
}

// Ioctl simplified ioct call
func ioctl(fd, op, arg uintptr) error {
	if _, _, ep := unix.Syscall(unix.SYS_IOCTL, fd, op, arg); ep != 0 {
		return syscall.Errno(ep)
	}
	return nil
}

const (
	ioctlSize     = uintptr(4)
	hciMaxDevices = 16
	typHCI        = 72 // 'H'
)

var (
	hciUpDevice      = ioW(typHCI, 201, ioctlSize) // HCIDEVUP
	hciDownDevice    = ioW(typHCI, 202, ioctlSize) // HCIDEVDOWN
	hciResetDevice   = ioW(typHCI, 203, ioctlSize) // HCIDEVRESET
	hciGetDeviceList = ioR(typHCI, 210, ioctlSize) // HCIGETDEVLIST
	hciGetDeviceInfo = ioR(typHCI, 211, ioctlSize) // HCIGETDEVINFO
)

type devRequest struct {
	id  uint16
	opt uint32
}

type devListRequest struct {
	devNum     uint16
	devRequest [hciMaxDevices]devRequest
}

type hciDevInfo struct {
	id         uint16
	name       [8]byte
	bdaddr     [6]byte
	flags      uint32
	devType    uint8
	features   [8]uint8
	pktType    uint32
	linkPolicy uint32
	linkMode   uint32
	aclMtu     uint16
	aclPkts    uint16
	scoMtu     uint16
	scoPkts    uint16

	stats hciDevStats
}

type hciDevStats struct {
	errRx  uint32
	errTx  uint32
	cmdTx  uint32
	evtRx  uint32
	aclTx  uint32
	aclRx  uint32
	scoTx  uint32
	scoRx  uint32
	byteRx uint32
	byteTx uint32
}

type skt struct {
	fd   int
	dev  int
	name string
	rmu  *sync.Mutex
	wmu  *sync.Mutex
}

// NewSocket ...
func NewSocket(n int, chk bool) (*skt, error) {
	fd, err := unix.Socket(unix.AF_BLUETOOTH, unix.SOCK_RAW, unix.BTPROTO_HCI)
	if err != nil {
		return nil, err
	}
	if n != -1 {
		return newSocket(fd, n, chk)
	}

	req := devListRequest{devNum: hciMaxDevices}
	if err := ioctl(uintptr(fd), hciGetDeviceList, uintptr(unsafe.Pointer(&req))); err != nil {
		return nil, err
	}
	for i := 0; i < int(req.devNum); i++ {
		d, err := newSocket(fd, i, chk)
		if err == nil {
			log.Printf("dev: %s opened", d.name)
			return d, err
		}
	}
	return nil, errors.New("no supported devices available")
}

func newSocket(fd, n int, chk bool) (*skt, error) {
	i := hciDevInfo{id: uint16(n)}
	if err := ioctl(uintptr(fd), hciGetDeviceInfo, uintptr(unsafe.Pointer(&i))); err != nil {
		return nil, err
	}
	name := string(i.name[:])
	// Check the feature list returned feature list.
	if chk && i.features[4]&0x40 == 0 {
		err := errors.New("does not support LE")
		log.Printf("dev: %s %s", name, err)
		return nil, err
	}
	log.Printf("dev: %s up", name)
	if err := ioctl(uintptr(fd), hciUpDevice, uintptr(n)); err != nil {
		if err != unix.EALREADY {
			return nil, err
		}
		log.Printf("dev: %s reset", name)
		if err := ioctl(uintptr(fd), hciResetDevice, uintptr(n)); err != nil {
			return nil, err
		}
	}
	log.Printf("dev: %s down", name)
	if err := ioctl(uintptr(fd), hciDownDevice, uintptr(n)); err != nil {
		return nil, err
	}

	// Attempt to use the linux 3.14 feature, if this fails with EINVAL fall back to raw access
	// on older kernels.
	sa := unix.SockaddrHCI{Dev: n, Channel: unix.HCI_CHANNEL_USER}
	if err := unix.Bind(fd, &sa); err != nil {
		if err != unix.EINVAL {
			return nil, err
		}
		log.Printf("dev: %s can't bind to hci user channel, err: %s.", name, err)
		sa := unix.SockaddrHCI{Dev: n, Channel: unix.HCI_CHANNEL_RAW}
		if err := unix.Bind(fd, &sa); err != nil {
			log.Printf("dev: %s can't bind to hci raw channel, err: %s.", name, err)
			return nil, err
		}
	}
	return &skt{
		fd:   fd,
		dev:  n,
		name: name,
		rmu:  &sync.Mutex{},
		wmu:  &sync.Mutex{},
	}, nil
}

func (d *skt) Read(b []byte) (int, error) {
	d.rmu.Lock()
	defer d.rmu.Unlock()
	return unix.Read(d.fd, b)
}

func (d *skt) Write(b []byte) (int, error) {
	d.wmu.Lock()
	defer d.wmu.Unlock()
	return unix.Write(d.fd, b)
}

func (d *skt) Close() error {
	return unix.Close(d.fd)
}
