package hci

import (
	"io"
	"log"
)

type cmdSender struct {
	skt   io.Writer
	sent  map[int]*cmdPkt
	chPkt chan *cmdPkt

	// Host to Controller command flow control [Vol 2, Part E, 4.4]
	chBufs chan []byte
}

func newCmdSender(skt io.Writer) *cmdSender {
	s := &cmdSender{
		skt:    skt,
		chPkt:  make(chan *cmdPkt),
		chBufs: make(chan []byte, 8),
		sent:   make(map[int]*cmdPkt),
	}
	go s.loop()
	return s
}

type cmdPkt struct {
	cmd  Command
	done chan []byte
}

func (s *cmdSender) send(c Command, r CommandRP) error {
	p := &cmdPkt{c, make(chan []byte)}
	s.chPkt <- p
	b := <-p.done
	if r == nil {
		return nil
	}
	return r.Unmarshal(b)
}

func (s *cmdSender) loop() {
	s.chBufs <- make([]byte, 64)

	for p := range s.chPkt {
		b := <-s.chBufs
		c := p.cmd
		b[0] = byte(pktTypeCommand) // HCI header
		b[1] = byte(c.OpCode())
		b[2] = byte(c.OpCode() >> 8)
		b[3] = byte(c.Len())
		if err := c.Marshal(b[4:]); err != nil {
			log.Printf("hci: failed to marshal cmd")
			return
		}

		s.sent[c.OpCode()] = p // TODO: lock
		if n, err := s.skt.Write(b[:4+c.Len()]); err != nil {
			log.Printf("hci: failed to send cmd")
		} else if n != 4+c.Len() {
			log.Printf("hci: failed to send whole cmd pkt to hci socket")
		}
	}
}
