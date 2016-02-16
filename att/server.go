package att

import (
	"bytes"
	"encoding/binary"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/currantlabs/bt/hci"
)

// Server implementas an ATT (Attribute Protocol) server.
type Server struct {
	l2c   hci.Conn
	ctx   context.Context
	attrs *Range

	// Refer to [Vol 3, Part F, 3.3.2 & 3.3.3] for the requirement of
	// sequential request-response protocol, and transactions.
	txBuf     []byte
	rxBuf     []byte
	chNotBuf  chan []byte
	chIndBuf  chan []byte
	chConfirm chan bool
}

// NewServer returns an ATT (Attribute Protocol) server.
func NewServer(ctx context.Context, a *Range, l2c hci.Conn, rxMTU int) *Server {
	// Although the rxBuf is initialized with the capacity of rxMTU, it is
	// not discovered, and only the default ATT_MTU (23 bytes) of it shall
	// be used until remote central request ExchangeMTU.
	s := &Server{
		l2c:   l2c,
		ctx:   ctx,
		attrs: a,

		txBuf:     make([]byte, 23, 23),
		rxBuf:     make([]byte, rxMTU, rxMTU),
		chNotBuf:  make(chan []byte, 1),
		chIndBuf:  make(chan []byte, 1),
		chConfirm: make(chan bool),
	}
	s.chNotBuf <- make([]byte, 23, 23)
	s.chIndBuf <- make([]byte, 23, 23)
	return s
}

// SendNotification sends notification to remote central.
func (s *Server) SendNotification(h uint16, data []byte) (int, error) {
	log.Printf("Notifying 0x%04X, %s", h, string(data))

	// Acquire and reuse notifyBuffer. Release it after usage.
	nBuf := <-s.chNotBuf
	defer func() { s.chNotBuf <- nBuf }()

	rsp := HandleValueNotification(nBuf)
	rsp.SetAttributeOpcode()
	rsp.SetAttributeHandle(h)
	buf := bytes.NewBuffer(rsp.AttributeValue())
	buf.Reset()
	if len(data) > buf.Cap() {
		data = data[:buf.Cap()]
	}
	buf.Write(data)
	return s.l2c.Write(rsp[:3+buf.Len()])
}

// SendIndication sends indication to remote central.
func (s *Server) SendIndication(h uint16, data []byte) (int, error) {
	log.Printf("Indicating 0x%04X, %s", h, string(data))

	// Acquire and reuse indicateBuffer. Release it after usage.
	iBuf := <-s.chIndBuf
	defer func() { s.chIndBuf <- iBuf }()

	rsp := HandleValueIndication(iBuf)
	rsp.SetAttributeOpcode()
	rsp.SetAttributeHandle(h)
	buf := bytes.NewBuffer(rsp.AttributeValue())
	buf.Reset()
	if len(data) > buf.Cap() {
		data = data[:buf.Cap()]
	}
	buf.Write(data)
	n, err := s.l2c.Write(rsp[:3+buf.Len()])
	if err != nil {
		return n, err
	}
	select {
	case <-s.chConfirm:
		return n, nil
	case <-time.After(time.Second * 30):
		return 0, ErrSeqProtoTimeout
	}
}

// Loop accepts incoming ATT request, and respond response.
func (s *Server) Loop() {
	for {
		n, err := s.l2c.Read(s.rxBuf)
		if n == 0 || err != nil {
			s.close()
			break
		}
		if rsp := s.handleRequest(s.rxBuf[:n]); rsp != nil {
			if len(rsp) != 0 {
				s.l2c.Write(rsp)
			}
		}
	}
}

func (s *Server) close() error {
	return s.l2c.Close()
}

func (s *Server) handleRequest(b []byte) []byte {
	var resp []byte
	log.Printf("req: % X", b)
	switch reqType := b[0]; reqType {
	case ExchangeMTURequestCode:
		resp = s.handleExchangeMTURequest(b)
	case FindInformationRequestCode:
		resp = s.handleFindInformationRequest(b)
	case FindByTypeValueRequestCode:
		resp = s.handleFindByTypeValueRequest(b)
	case ReadByTypeRequestCode:
		resp = s.handleReadByTypeRequest(b)
	case ReadRequestCode:
		resp = s.handleReadRequest(b)
	case ReadBlobRequestCode:
		resp = s.handleReadBlobRequest(b)
	case ReadByGroupTypeRequestCode:
		resp = s.handleReadByGroupRequest(b)
	case WriteRequestCode:
		resp = s.handleWriteRequest(b)
	case WriteCommandCode:
		s.handleWriteCommand(b)
	case ReadMultipleRequestCode,
		PrepareWriteRequestCode,
		ExecuteWriteRequestCode,
		SignedWriteCommandCode:
		fallthrough
	case HandleValueConfirmationCode:
		select {
		case s.chConfirm <- true:
		default:
			log.Printf("recieved a spurious confirmation")
		}
	default:
		resp = NewErrorResponse(reqType, 0x0000, ErrReqNotSupp)
	}
	log.Printf("resp: % X", resp)
	return resp
}

// handle MTU Exchange request. [Vol 3, Part F, 3.4.2]
func (s *Server) handleExchangeMTURequest(r ExchangeMTURequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 3:
		fallthrough
	case r.ClientRxMTU() < 23:
		return NewErrorResponse(r.AttributeOpcode(), 0x0000, ErrInvalidPDU)
	}

	txMTU := int(r.ClientRxMTU())
	s.l2c.SetTxMTU(txMTU)
	s.l2c.SetRxMTU(len(s.rxBuf))

	if txMTU != len(s.txBuf) {
		// Apply the txMTU afer this response has been sent and before
		// any other attribute protocol PDU is sent.
		defer func() {
			s.txBuf = make([]byte, txMTU, txMTU)
			<-s.chNotBuf
			s.chNotBuf <- make([]byte, txMTU, txMTU)
			<-s.chIndBuf
			s.chIndBuf <- make([]byte, txMTU, txMTU)
		}()
	}

	rsp := ExchangeMTUResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	rsp.SetServerRxMTU(uint16(len(s.rxBuf)))
	return rsp[:3]
}

// handle Find Information request. [Vol 3, Part F, 3.4.3.1 & 3.4.3.2]
func (s *Server) handleFindInformationRequest(r FindInformationRequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 5:
		return NewErrorResponse(r.AttributeOpcode(), 0x0000, ErrInvalidPDU)
	case r.StartingHandle() == 0 || r.StartingHandle() > r.EndingHandle():
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrInvalidHandle)
	}

	rsp := FindInformationResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	rsp.SetFormat(0x00)
	buf := bytes.NewBuffer(rsp.InformationData())
	buf.Reset()

	// Each response shall contain Types of the same format.
	for _, a := range s.attrs.Subrange(r.StartingHandle(), r.EndingHandle()) {
		if rsp.Format() == 0 {
			rsp.SetFormat(0x01)
			if a.Type.Len() == 16 {
				rsp.SetFormat(0x02)
			}
		}
		if rsp.Format() == 0x01 && a.Type.Len() != 2 {
			break
		}
		if rsp.Format() == 0x02 && a.Type.Len() != 16 {
			break
		}
		if buf.Len()+2+a.Type.Len() > buf.Cap() {
			break
		}
		binary.Write(buf, binary.LittleEndian, a.Handle)
		binary.Write(buf, binary.LittleEndian, a.Type)
	}

	// Nothing has been found.
	if rsp.Format() == 0 {
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrAttrNotFound)
	}
	return rsp[:2+buf.Len()]
}

// handle Find By Type Value request. [Vol 3, Part F, 3.4.3.3 & 3.4.3.4]
func (s *Server) handleFindByTypeValueRequest(r FindByTypeValueRequest) []byte {
	// Validate the request.
	switch {
	case len(r) < 7:
		return NewErrorResponse(r.AttributeOpcode(), 0x0000, ErrInvalidPDU)
	case r.StartingHandle() == 0 || r.StartingHandle() > r.EndingHandle():
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrInvalidHandle)
	}

	rsp := FindByTypeValueResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.HandleInformationList())
	buf.Reset()

	for _, a := range s.attrs.Subrange(r.StartingHandle(), r.EndingHandle()) {
		v, starth, endh := a.Value, a.Handle, a.EndingHandle
		if v == nil {
			// The value shall not exceed ATT_MTU - 7 bytes.
			// Since ResponseWriter caps the value at the capacity,
			// we allocate one extra byte, and the written length.
			buf2 := bytes.NewBuffer(make([]byte, 0, len(s.txBuf)-7+1))
			e := a.Pvt.Handle(s.ctx, r, &ResponseWriter{buf: buf2})
			if e != ErrSuccess || buf2.Len() > len(s.txBuf)-7 {
				return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrInvalidHandle)
			}
			endh = a.Handle
		}
		if !(UUID(v).Equal(UUID(r.AttributeValue()))) {
			continue
		}
		if buf.Len()+4 > buf.Cap() {
			break
		}
		binary.Write(buf, binary.LittleEndian, starth)
		binary.Write(buf, binary.LittleEndian, endh)
	}
	if buf.Len() == 0 {
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrAttrNotFound)
	}

	return rsp[:1+buf.Len()]
}

// handle Read By Type request. [Vol 3, Part F, 3.4.4.1 & 3.4.4.2]
func (s *Server) handleReadByTypeRequest(r ReadByTypeRequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 7 && len(r) != 21:
		return NewErrorResponse(r.AttributeOpcode(), 0x0000, ErrInvalidPDU)
	case r.StartingHandle() == 0 || r.StartingHandle() > r.EndingHandle():
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrInvalidHandle)
	}

	rsp := ReadByTypeResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.AttributeDataList())
	buf.Reset()

	// handle length (2 bytes) + value length.
	// Each response shall only contains values with the same size.
	dlen := 0
	for _, a := range s.attrs.Subrange(r.StartingHandle(), r.EndingHandle()) {
		if !a.Type.Equal(UUID(r.AttributeType())) {
			continue
		}
		v := a.Value
		if v == nil {
			buf2 := bytes.NewBuffer(make([]byte, 0, len(s.txBuf)-2))
			if e := a.Pvt.Handle(s.ctx, r, &ResponseWriter{buf: buf2}); e != ErrSuccess {
				// Return if the first value read cause an error.
				if dlen == 0 {
					return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), e)
				}
				// Otherwise, skip to the next one.
				break
			}
			v = buf2.Bytes()
		}
		if dlen == 0 {
			// Found the first value.
			dlen = 2 + len(v)
			if dlen > 255 {
				dlen = 255
			}
			if dlen > buf.Cap() {
				dlen = buf.Cap()
			}
			rsp.SetLength(uint8(dlen))
		} else if 2+len(v) != dlen {
			break
		}
		binary.Write(buf, binary.LittleEndian, a.Handle)
		binary.Write(buf, binary.LittleEndian, v[:dlen-2])
	}
	if dlen == 0 {
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrAttrNotFound)
	}
	return rsp[:2+buf.Len()]
}

// handle Read request. [Vol 3, Part F, 3.4.4.3 & 3.4.4.4]
func (s *Server) handleReadRequest(r ReadRequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 3:
		return NewErrorResponse(r.AttributeOpcode(), 0x0000, ErrInvalidPDU)
	}

	rsp := ReadResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.AttributeValue())
	buf.Reset()

	a, ok := s.attrs.At(r.AttributeHandle())
	if !ok {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ErrInvalidHandle)
	}

	// Simple case. Read-only, no-authorization, no-authentication.
	if a.Value != nil {
		binary.Write(buf, binary.LittleEndian, a.Value)
		return rsp[:1+buf.Len()]
	}

	// Pass the request to upper layer with the ResponseWriter, which caps
	// the buffer to a valid length of payload.
	if e := a.Pvt.Handle(s.ctx, r, &ResponseWriter{buf: buf}); e != ErrSuccess {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), e)
	}
	return rsp[:1+buf.Len()]
}

// handle Read Blob request. [Vol 3, Part F, 3.4.4.5 & 3.4.4.6]
func (s *Server) handleReadBlobRequest(r ReadBlobRequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 5:
		return NewErrorResponse(r.AttributeOpcode(), 0x0000, ErrInvalidPDU)
	}

	a, ok := s.attrs.At(r.AttributeHandle())
	if !ok {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ErrInvalidHandle)
	}

	rsp := ReadBlobResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.PartAttributeValue())
	buf.Reset()

	// Simple case. Read-only, no-authorization, no-authentication.
	if a.Value != nil {
		binary.Write(buf, binary.LittleEndian, a.Value)
		return rsp[:1+buf.Len()]
	}

	// Pass the request to upper layer with the ResponseWriter, which caps
	// the buffer to a valid length of payload.
	if e := a.Pvt.Handle(s.ctx, r, &ResponseWriter{buf: buf}); e != ErrSuccess {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), e)
	}
	return rsp[:1+buf.Len()]
}

// handle Read Blob request. [Vol 3, Part F, 3.4.4.9 & 3.4.4.10]
func (s *Server) handleReadByGroupRequest(r ReadByGroupTypeRequest) []byte {
	// Validate the request.
	switch {
	case len(r) != 7 && len(r) != 21:
		return NewErrorResponse(r.AttributeOpcode(), 0x0000, ErrInvalidPDU)
	case r.StartingHandle() == 0 || r.StartingHandle() > r.EndingHandle():
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrInvalidHandle)
	}

	rsp := ReadByGroupTypeResponse(s.txBuf)
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.AttributeDataList())
	buf.Reset()

	dlen := 0
	for _, a := range s.attrs.Subrange(r.StartingHandle(), r.EndingHandle()) {
		v := a.Value
		if v == nil {
			buf2 := bytes.NewBuffer(make([]byte, buf.Cap()-buf.Len()-4))
			if e := a.Pvt.Handle(s.ctx, r, &ResponseWriter{buf: buf2}); e != ErrSuccess {
				return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), e)
			}
			v = buf2.Bytes()
		}
		if dlen == 0 {
			dlen = 4 + len(v)
			if dlen > 255 {
				dlen = 255
			}
			if dlen > buf.Cap() {
				dlen = buf.Cap()
			}
			rsp.SetLength(uint8(dlen))
		} else if 4+len(v) != dlen {
			break
		}
		binary.Write(buf, binary.LittleEndian, a.Handle)
		binary.Write(buf, binary.LittleEndian, a.EndingHandle)
		binary.Write(buf, binary.LittleEndian, v[:dlen-4])
	}
	if dlen == 0 {
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrAttrNotFound)
	}
	return rsp[:2+buf.Len()]
}

// handle Write request. [Vol 3, Part F, 3.4.5.1 & 3.4.5.2]
func (s *Server) handleWriteRequest(r WriteRequest) []byte {
	// Validate the request.
	switch {
	case len(r) < 3:
		return NewErrorResponse(r.AttributeOpcode(), 0x0000, ErrInvalidPDU)
	}

	a, ok := s.attrs.At(r.AttributeHandle())
	if !ok {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ErrInvalidHandle)
	}

	// We don't support write to static value. Pass the request to upper layer.
	if a.Pvt == nil {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ErrWriteNotPerm)
	}
	if e := a.Pvt.Handle(s.ctx, r, nil); e != ErrSuccess {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), e)
	}
	return []byte{WriteResponseCode}
}

// handle Write command. [Vol 3, Part F, 3.4.5.3]
func (s *Server) handleWriteCommand(r WriteCommand) []byte {
	// Validate the request.
	switch {
	case len(r) <= 3:
		return nil
	}

	a, ok := s.attrs.At(r.AttributeHandle())
	if !ok {
		return nil
	}

	// We don't support write to static value. Pass the request to upper layer.
	if a.Pvt == nil {
		return nil
	}
	if e := a.Pvt.Handle(s.ctx, r, nil); e != ErrSuccess {
		return nil
	}
	return nil
}