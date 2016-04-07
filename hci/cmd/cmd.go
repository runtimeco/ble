//go:generate sh -c "go run ../tools/codegen/codegen.go -tmpl cmd -in ../tools/codegen/cmd.json -out cmd_gen.go && goimports -w cmd_gen.go"

package cmd

import (
	"bytes"
	"encoding/binary"
	"io"
)

type command interface {
	Len() int
}

type commandRP interface {
	Unmarshal(b []byte) error
}

func marshal(c command, b []byte) error {
	buf := bytes.NewBuffer(b)
	buf.Reset()
	if buf.Cap() < c.Len() {
		return io.ErrShortBuffer
	}
	return binary.Write(buf, binary.LittleEndian, c)
}

func unmarshal(c commandRP, b []byte) error {
	buf := bytes.NewBuffer(b)
	return binary.Read(buf, binary.LittleEndian, c)
}
