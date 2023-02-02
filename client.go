package rtmp

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	RTMP_VERSION = 3
)

func NewClient() *client {
	return &client{
		rChunkSize:     128,
		wChunkSize:     128,
		rChunkStream:   make(map[uint32]chunkReader),
		wChunkStream:   make(map[uint32]chunkWriter),
		peerWindowSize: 0xFFFFFFFF,
	}
}

type client struct {
	sync.Mutex
	conn           net.Conn
	bufr           *bufio.Reader
	bufw           *bufio.Writer
	host           string
	rChunkSize     uint32                 // 读chunk大小
	wChunkSize     uint32                 // 写chunk大小
	rChunkStream   map[uint32]chunkReader // read chunk stream
	wChunkStream   map[uint32]chunkWriter // write chunk stream
	wSequence      uint32
	rSequence      uint32
	lastRSequence  uint32
	windowSize     uint32 // 窗口大小
	peerWindowSize uint32 // 对方窗口大小
	err            error
}

func (c *client) Err() error {
	return c.err
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

func (c *client) WriteByte(b byte) (err error) {
	if err = c.bufw.WriteByte(b); err != nil {
		atomic.AddUint32(&c.wSequence, 1)
	}
	return
}

func (c *client) Flush() error {
	return c.bufw.Flush()
}

func (c *client) Dail(addr string) *client {
	if c.err != nil {
		return c
	}
	if addr == "" {
		c.host = "localhost:1935"
		addr = ":1935"
	} else if addr[0] == ':' {
		c.host = "localhost" + addr
	} else if !strings.Contains(addr, ":") {
		c.host = addr
		addr = addr + ":1935"
	}
	c.conn, c.err = net.DialTimeout("tcp", addr, time.Second*5)
	if c.err == nil {
		c.bufr = bufio.NewReader(c.conn)
		c.bufw = bufio.NewWriter(c.conn)
	}
	return c
}

func (c *client) Close() {
	if c.conn == nil {
		return
	}
	c.Flush()
	err := c.conn.Close()
	if err != nil {
		log.Printf("close rtmp client error: %v", err)
	}
}

func (c *client) Handshake() *client {
	if c.err != nil {
		return c
	}
	c0c1 := make([]byte, 1537)
	c0c1[0] = RTMP_VERSION
	for i := 9; i < 1537; i += 8 {
		binary.BigEndian.PutUint64(c0c1[i:i+8], rand.Uint64())
	}
	// write c0c1
	if _, c.err = c.Write(c0c1); c.err != nil {
		return c
	}
	if c.err = c.Flush(); c.err != nil {
		return c
	}
	// read s0s1
	if _, c.err = c.Read(c0c1); c.err != nil {
		return c
	}
	// write c2
	c2 := c0c1[1:]
	if _, c.err = c.Write(c2); c.err != nil {
		return c
	}
	if c.err = c.Flush(); c.err != nil {
		return c
	}
	// read s2
	_, c.err = c.Read(c2)
	return c
}

func (c *client) ReadMessage() (Message, error) {
START:
	tmf, csid, err := readBasicHeader(c) // read Basic Header
	if err != nil {
		return Message{}, err
	}
	// get chunk stream
	cs, ok := c.rChunkStream[csid]
	if !ok {
		// create a chunk stream
		cs = newChunkReader(c, csid)
		c.rChunkStream[csid] = cs
	}
	// read chunk
	if err = cs.readChunk(tmf, c.rChunkSize); err != nil {
		return Message{}, err
	}
	if !cs.complete() {
		goto START //消息未读完，继续读下一个chunk
	}

	msg := cs.assembleMessage()
	cs.reset() // reset for the next message

	// send window acknowledgement
	if c.rSequence-c.lastRSequence >= c.peerWindowSize {
		c.WriteMessage(AcknowledgementMessage{c.rSequence})
	}

	// handle message
	switch msg.tid {
	case 1, 2, 3, 5, 6:
		// protocol control message
		if err = c.handleProtocol(msg); err != nil {
			return Message{}, fmt.Errorf("handle protocol error: %w", err)
		}
		goto START
	default:
		return Message{
			Header: MessageHeader{
				Type:      msg.tid,
				Length:    msg.length,
				Timestamp: msg.timestamp,
				StreamID:  cs.(*chunkStream).msid,
			},
			Payload: msg.payload,
		}, nil
	}
}

func (c *client) WriteMessage(m Messager) *client {
	if m == nil || c.err != nil {
		return c
	}

	c.Lock()
	defer c.Unlock()

	mtid := m.Tid()
	csid, msid := getCsidAndMsid(mtid)
	cs, ok := c.wChunkStream[csid]
	if !ok {
		// create chunk stream
		cs = newChunkWriter(c, csid)
		c.wChunkStream[csid] = cs
	}
	if c.err = cs.writeMessage(m, msid, c.wChunkSize); c.err != nil {
		return c
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
	return c
}

func (c *client) handleProtocol(msg *message) (err error) {
	switch msg.tid {
	case 1: // Protocol Control Message: Set Chunk Size
		var m SetChunkSizeMessage
		if err = m.Unmarshal(msg.payload); err != nil {
			return
		}
		c.rChunkSize = m.chunkSize

	case 2: // Protocol Control Message: Abort Message
		var m AbortMessage
		if err = m.Unmarshal(msg.payload); err != nil {
			return
		}
		if cs, ok := c.rChunkStream[m.chunkStreamId]; ok {
			cs.reset()
		}

	case 3: // Protocol Control Message: Acknowledgement
		var m AcknowledgementMessage
		if err = m.Unmarshal(msg.payload); err != nil {
			return
		}
		// client has received m.sequenceNumber
		// Log("write no ack: %d", c.wSequence-m.sequenceNumber)

	case 5: // Protocol Control Message: Window Acknowledgement Size
		var m WindowAcknowledgementSizeMessage
		if err = m.Unmarshal(msg.payload); err != nil {
			return
		}
		c.peerWindowSize = m.windowSize

	case 6: // Protocol Control Message: Set Peer Bandwidth
		var m SetPeerBandwidthMesage
		if err = m.Unmarshal(msg.payload); err != nil {
			return
		}
		c.peerWindowSize = m.windowSize
	}
	return nil
}

func (c *client) SetChunkSize(size uint32) *client {
	if c.err != nil {
		return c
	}
	return c.WriteMessage(SetChunkSizeMessage{size})
}

func (c *client) Abort(csid uint32) *client {
	if c.err != nil {
		return c
	}
	return c.WriteMessage(AbortMessage{csid})
}

func (c *client) SetWindowSize(size uint32) *client {
	if c.err != nil {
		return c
	}
	return c.WriteMessage(WindowAcknowledgementSizeMessage{size})
}

func (c *client) SetBandwidth(size uint32, mode uint8) *client {
	if c.err != nil {
		return c
	}
	return c.WriteMessage(SetPeerBandwidthMesage{size, mode})
}

func (c *client) StreamBegin(msid uint32) *client {
	if c.err != nil {
		return c
	}
	return c.WriteMessage(UserControlMessage{EventType: STREAM_BEGIN, Param1: msid})
}

func (c *client) Connect(app string) *client {
	if c.err != nil {
		return c
	}
	msg := CommandMessage{
		Name:          "connect",
		TransactionId: 1,
		arr: []any{map[string]any{
			"app":           app,
			"flashVer":      "LNX 9,0,124,2",
			"tcUrl":         "rtmp://" + c.host + "/" + app,
			"capabilities":  15,
			"audioCodecs":   0x80,
			"videoCodecs":   0x40,
			"videoFunction": 1,
		}},
	}
	return c.WriteMessage(msg)
}

func (c *client) CreateStream(msid uint32) *client {
	if c.err != nil {
		return c
	}
	msg := CommandMessage{
		Name:          "createStream",
		TransactionId: 2,
		arr:           []any{nil},
	}
	return c.WriteMessage(msg)
}

func (c *client) Publish(streamName string) *client {
	if c.err != nil {
		return c
	}
	msg := CommandMessage{
		Name:          "publish",
		TransactionId: 5,
		arr:           []any{nil, streamName, "live"},
	}
	return c.WriteMessage(msg)
}

func (c *client) Unpublish(streamName string) *client {
	if c.err != nil {
		return c
	}
	msg := CommandMessage{
		Name:          "FCUpublish",
		TransactionId: 6,
		arr:           []any{nil, streamName},
	}
	return c.WriteMessage(msg)
}

func (c *client) Data(timestamp uint32, data []byte) *client {
	if c.err != nil {
		return c
	}
	msg := message{
		tid:       18,
		timestamp: timestamp,
		payload:   data,
	}
	return c.WriteMessage(msg)
}

func (c *client) Audio(timestamp uint32, data []byte) *client {
	if c.err != nil {
		return c
	}
	msg := message{
		tid:       8,
		timestamp: timestamp,
		payload:   data,
	}
	return c.WriteMessage(msg)
}

func (c *client) Video(timestamp uint32, data []byte) *client {
	if c.err != nil {
		return c
	}
	msg := message{
		tid:       9,
		timestamp: timestamp,
		payload:   data,
	}
	return c.WriteMessage(msg)
}
