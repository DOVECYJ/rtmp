//go:build client
// +build client

package rtmp

import (
	"net/http"
	"bufio"
	"io"
	"net"
	"sync/atomic"
	"time"
)

func NewClient() *client {
	return &client{
		rChunkSize:   128,
		wChunkSize:   128,
		rChunkStream: make(map[uint32]chunkReader),
		wChunkStream: make(map[uint32]chunkWriter),
	}
}

type client struct {
	conn           net.Conn
	bufr           *bufio.Reader
	bufw           *bufio.Writer
	msgChan        chan *message
	rChunkSize     uint32                 // 读chunk大小
	wChunkSize     uint32                 // 写chunk大小
	rChunkStream   map[uint32]chunkReader // read chunk stream
	wChunkStream   map[uint32]chunkWriter // write chunk stream
	wSequence      uint32
	rSequence      uint32
	lastRSequence  uint32
	windowSize     uint32 // 窗口大小
	peerWindowSize uint32 // 对方窗口大小
}

// implement io.Reader
func (c *client) Read(p []byte) (n int, err error) {
	n, err = io.ReadFull(c.bufr, p)
	atomic.AddUint32(&c.rSequence, uint32(n))
	return
}

// implement io.Writer
func (c *client) Write(p []byte) (n int, err error) {
	n, err = c.bufw.Write(p)
	atomic.AddUint32(&c.wSequence, uint32(n))
	return
}

func (c *client) Flush() error {
	return c.bufw.Flush()
}

func (c *client) Dail(addr string) (ch <-chan *message, err error) {
	c.conn, err = net.DialTimeout("tcp", addr, time.Second*5)
	if err != nil {
		return
	}
	c.bufr = bufio.NewReader(c.conn)
	c.bufw = bufio.NewWriter(c.conn)
	c.msgChan = make(chan *message, 128)
	go func() {
		for {

		}
	}()
	return
}

func (c *client) Handshake() error {
	return handshake(c)
}

func (c *client) ReadMessage() (Message, error) {
	http.Get()
}

func (c *client) WriteMessage(m Messager) error {
	if m == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	mtid := m.Tid()
	csid, msid := getCsidAndMsid(mtid)
	cs, ok := c.wChunkStream[csid]
	if !ok {
		// create chunk stream
		cs = newChunkWriter(c, csid)
		c.wChunkStream[csid] = cs
	}
	if err = cs.writeMessage(m, msid, c.wChunkSize); err != nil {
		return
	}
	// 处理协议控制消息
	switch m := m.(type) {
	case SetChunkSizeMessage:
		c.wChunkSize = m.chunkSize
	case AbortMessage:
		if cs, ok := c.wChunkStream[m.chunkStreamId]; ok {
			cs.discard()
		}
	case AcknowledgementMessage:
		c.lastRSequence = c.rSequence
	case WindowAcknowledgementSizeMessage:
		c.windowSize = m.windowSize
	case SetPeerBandwidthMesage:
		c.windowSize = m.windowSize
	}
	return
}
