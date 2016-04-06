package buffer

import "sync"

// Buffer ...
type Buffer []byte

// Pool ...
type Pool struct {
	sync.Mutex

	size int
	cnt  int
	ch   chan Buffer
}

// NewPool ...
func NewPool(size int, cnt int) *Pool {
	ch := make(chan Buffer, cnt)
	for len(ch) < cnt {
		ch <- make([]byte, size)
	}
	return &Pool{size: size, cnt: cnt, ch: ch}
}

// Client ...
type Client struct {
	p    *Pool
	used chan Buffer
}

// NewClient ...
func NewClient(p *Pool) *Client {
	return &Client{p: p, used: make(chan Buffer, p.cnt)}
}

// Lock ...
func (c *Client) Lock() { c.p.Lock() }

// Unlock ...
func (c *Client) Unlock() { c.p.Unlock() }

// Alloc ...
func (c *Client) Alloc() Buffer {
	b := <-c.p.ch
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
