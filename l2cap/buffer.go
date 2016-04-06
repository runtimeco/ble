package l2cap

import (
	"bytes"
	"sync"
)

// Pool ...
type Pool struct {
	sync.Mutex

	size int
	cnt  int
	ch   chan *bytes.Buffer
}

// NewPool ...
func NewPool(size int, cnt int) *Pool {
	ch := make(chan *bytes.Buffer, cnt)
	for len(ch) < cnt {
		ch <- bytes.NewBuffer(make([]byte, size))
	}
	return &Pool{size: size, cnt: cnt, ch: ch}
}

// Client ...
type Client struct {
	p    *Pool
	used chan *bytes.Buffer
}

// NewClient ...
func NewClient(p *Pool) *Client {
	return &Client{p: p, used: make(chan *bytes.Buffer, p.cnt)}
}

// Lock ...
func (c *Client) Lock() { c.p.Lock() }

// Unlock ...
func (c *Client) Unlock() { c.p.Unlock() }

// Alloc ...
func (c *Client) Alloc() *bytes.Buffer {
	b := <-c.p.ch
	b.Reset()
	c.used <- b
	return b
}

// Free ...
func (c *Client) Free() {
	b := <-c.used
	c.p.ch <- b
}

// FreeAll ...
func (c *Client) FreeAll() {
	for b := range c.used {
		c.p.ch <- b
	}
}
