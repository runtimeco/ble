package hci

import "io"

type aclProcessor struct {
	skt           io.Writer
	bufSize       int
	bufCnt        int
	handleACLData func([]byte)
}

func newACLProcessor(skt io.Writer) *aclProcessor {
	return &aclProcessor{
		skt: skt,
	}
}

func (a *aclProcessor) setACLProcessor(f func([]byte)) (w io.Writer, size int, cnt int) {
	a.handleACLData = f
	return a.skt, a.bufSize, a.bufCnt
}
