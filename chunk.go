package rtmp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

type msgHead struct {
	mtid      uint8  // message type id
	timestamp uint32 //时间戳
	length    uint32 //消息长度
	msid      uint32 //message stream id
	delta     uint32 //timestamp delta
}

//  0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                    timestamp                  |message length |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |    message length (cont)      |message type id| msg stream id |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |           message stream id (cont)            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
func (m *msgHead) readFmt0(r io.Reader) (err error) {
	bs := make([]byte, 11)
	if _, err = r.Read(bs); err != nil {
		return
	}
	msid := m.msid
	m.timestamp = uint32(bs[0])<<16 | uint32(bs[1])<<8 | uint32(bs[2])
	m.length = uint32(bs[3])<<16 | uint32(bs[4])<<8 | uint32(bs[5])
	m.mtid = uint8(bs[6])
	m.msid = binary.LittleEndian.Uint32(bs[7:]) // Message stream ID is stored in little-endian format
	// 读取扩展时间戳
	if m.timestamp == 0xFFFFFF {
		bs = bs[:4]
		if _, err = r.Read(bs); err != nil {
			return
		}
		m.timestamp = binary.BigEndian.Uint32(bs)
	}
	m.delta = 0
	if m.msid != msid {
		Log("message stream(%d) has been replaced by message stream(%d)", msid, m.msid)
	}
	return
}

func (m *msgHead) writeFmt0(w io.Writer) error {
	var bs []byte
	if m.timestamp < 0xFFFFFF {
		bs = make([]byte, 11)
		bs[0] = byte(m.timestamp >> 16)
		bs[1] = byte(m.timestamp >> 8)
		bs[2] = byte(m.timestamp)
	} else {
		bs = make([]byte, 15)
		bs[0], bs[1], bs[2] = 0xFF, 0xFF, 0xFF
		binary.BigEndian.PutUint32(bs[11:], m.timestamp)
	}
	bs[3] = byte(m.length >> 16)
	bs[4] = byte(m.length >> 8)
	bs[5] = byte(m.length)
	bs[6] = m.mtid
	binary.LittleEndian.PutUint32(bs[7:11], m.msid) // message stream id是小端序
	n, err := w.Write(bs)
	if err != nil {
		return err
	}
	if n != len(bs) {
		return fmt.Errorf("write message header 0 error, write %d of %d bytes", n, len(bs))
	}
	m.delta = 0
	return nil
}

//  0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                 timestamp delta               |message length |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |    message length (cont)      |message type id|
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
func (m *msgHead) readFmt1(r io.Reader) (err error) {
	bs := make([]byte, 7)
	if _, err = r.Read(bs); err != nil {
		return
	}
	m.delta = uint32(bs[0])<<16 | uint32(bs[1])<<8 | uint32(bs[2])
	m.length = uint32(bs[3])<<16 | uint32(bs[4])<<8 | uint32(bs[5])
	m.mtid = bs[6]
	// 读取扩展时间增量
	if m.delta == 0xFFFFFF {
		bs = bs[:4]
		if _, err = r.Read(bs); err != nil {
			return
		}
		m.delta = binary.BigEndian.Uint32(bs)
	}
	m.timestamp += m.delta
	return
}

func (m *msgHead) writeFmt1(w io.Writer) error {
	var bs []byte
	if m.delta < 0xFFFFFF {
		bs = make([]byte, 7)
		bs[0] = byte(m.delta >> 16)
		bs[1] = byte(m.delta >> 8)
		bs[2] = byte(m.delta)
	} else {
		bs = make([]byte, 11)
		bs[0], bs[1], bs[2] = 0xFF, 0xFF, 0xFF
		binary.BigEndian.PutUint32(bs[7:], m.delta)
	}
	bs[3] = byte(m.length >> 16)
	bs[4] = byte(m.length >> 8)
	bs[5] = byte(m.length)
	bs[6] = m.mtid
	n, err := w.Write(bs)
	if err != nil {
		return err
	}
	if n != len(bs) {
		return fmt.Errorf("write message header 1 error, write %d of %d bytes", n, len(bs))
	}
	return nil
}

//  0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                 timestamp delta               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
func (m *msgHead) readFmt2(r io.Reader) (err error) {
	bs := make([]byte, 3, 4)
	if _, err = r.Read(bs); err != nil {
		return
	}
	m.delta = uint32(bs[0])<<16 | uint32(bs[1])<<8 | uint32(bs[2])
	// 读取扩展时间增量
	if m.delta == 0xFFFFFF {
		bs = bs[:4]
		if _, err = r.Read(bs); err != nil {
			return
		}
		m.delta = binary.BigEndian.Uint32(bs)
	}
	m.timestamp += m.delta
	return
}

func (m *msgHead) writeFmt2(w io.Writer) error {
	var bs []byte
	if m.delta < 0xFFFFFF {
		bs = make([]byte, 3)
		bs[0] = byte(m.delta >> 16)
		bs[1] = byte(m.delta >> 8)
		bs[2] = byte(m.delta)
	} else {
		bs = make([]byte, 7)
		bs[0], bs[1], bs[2] = 0xFF, 0xFF, 0xFF
		binary.BigEndian.PutUint32(bs[3:], m.delta)
	}
	n, err := w.Write(bs)
	if err != nil {
		return err
	}
	if n != len(bs) {
		return fmt.Errorf("write message header 2 error, write %d of %d bytes", n, len(bs))
	}
	return nil
}

// 0 byte for message header 3
func (m *msgHead) readFmt3() {
	if m.delta == 0 {
		return
	}
	m.timestamp += m.delta
	// ExternTimestamp is present in Type 3 chunks when the most recent
	// Type 0, 1, or 2 chunk for the same chunk stream ID indicated
	// the presence of an extended timestamp field.
	// Type 3连timestamp都没有，怎么会有extern timestamp ？
}

//  0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
// +-+-+-+-+-+-+-+-+
// |fmt|   cs id   |
// +-+-+-+-+-+-+-+-+
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |fmt|     0     |   cs id - 64  |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |fmt|     1     |           cs id - 64          |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// read Basic Header
func readBasicHeader(r io.Reader) (tmf uint8, csid uint32, err error) {
	bs := make([]byte, 1, 2)
	if _, err = r.Read(bs); err != nil {
		return
	}

	tmf = uint8(bs[0] >> 6)
	csid = uint32(bs[0] & 0x3f)
	if csid == 0 {
		if _, err = r.Read(bs); err != nil {
			return
		}
		csid = uint32(bs[0]) + 64
	} else if csid == 1 {
		bs = bs[:2]
		if _, err = r.Read(bs); err != nil {
			return
		}
		// bs[1]*256 + bs[0]
		csid = uint32(binary.LittleEndian.Uint16(bs)) + 64
	}
	return
}

// write Basic Header
func writeBasicHeader(w io.Writer, tmf uint8, csid uint32) error {
	var bs []byte
	if csid < 64 {
		bs = make([]byte, 1)
		bs[0] = tmf<<6 | uint8(csid)
	} else if csid < 320 {
		bs := make([]byte, 2)
		bs[0] = tmf << 6
		bs[1] = byte(csid - 64)
	} else {
		bs = make([]byte, 3)
		bs[0] = tmf<<6 | 1
		binary.LittleEndian.PutUint16(bs[1:], uint16(csid-64))
	}
	n, err := w.Write(bs)
	if err != nil {
		return err
	}
	if n != len(bs) {
		return fmt.Errorf("write basic header error, write %d of %d bytes", n, len(bs))
	}
	return nil
}

type chunkReader interface {
	reset()
	readChunk(tmf uint8, chunkSize uint32) error
	assembleMessage() *message
	complete() bool
}

type chunkWriter interface {
	writeMessage(m Messager, msid, chunkSize uint32) error
	discard()
}

func newChunkReader(rw ReadWriteFlusher, csid uint32) chunkReader {
	return &chunkStream{
		done:    true,
		id:      csid,
		payload: make([]byte, 0, 512),
		rw:      rw,
	}
}

func newChunkWriter(rw ReadWriteFlusher, csid uint32) chunkWriter {
	return &chunkStream{
		id: csid,
		rw: rw,
	}
}

// message stream id -> message stream
//
// chunk stream id |-> message stream
//                 |-> message stream
// the rtmp chunk stream, messages are transmitted over it
type chunkStream struct {
	msgHead        // message header in chunk
	done    bool   // complete for read message and inited for write
	id      uint32 // message stream id
	index   uint32 // received for read and sended for write
	payload []byte // chunk payload
	rw      ReadWriteFlusher
}

// reset chunk stream for next read
func (cs *chunkStream) reset() {
	cs.done = true
	cs.index = 0
	cs.payload = cs.payload[:0]
}

func (cs *chunkStream) readChunk(tmf uint8, chunkSize uint32) (err error) {
	// read Message Header and Extern Timestamp
	if err = cs.readMessageHeader(tmf); err != nil {
		return
	}
	// read chunk payload
	return cs.readChunkPayload(chunkSize)
}

// read Message Header
func (cs *chunkStream) readMessageHeader(tmf uint8) (err error) {
	switch tmf {
	case 0:
		err = cs.msgHead.readFmt0(cs.rw)
	case 1:
		err = cs.msgHead.readFmt1(cs.rw)
	case 2:
		err = cs.msgHead.readFmt2(cs.rw)
	case 3:
		if cs.done { // 决定要不要加timestamp delta
			cs.msgHead.readFmt3()
		}
	}
	return
}

// read the payload of a chunk
func (cs *chunkStream) readChunkPayload(maxSize uint32) (err error) {
	// prepare memory for message payload
	if cs.done {
		if cap(cs.payload) < int(cs.length) {
			cs.payload = make([]byte, cs.length)
		} else {
			cs.payload = cs.payload[:cs.length]
		}
	}
	remain := cs.length - cs.index
	if remain > maxSize {
		remain = maxSize
	}
	// read payload
	part := cs.payload[cs.index : cs.index+remain]
	if _, err = cs.rw.Read(part); err != nil {
		return
	}
	cs.index += remain
	if cs.index > cs.length {
		err = fmt.Errorf("received(%d) over than length(%d)", cs.index, cs.length)
	} else {
		cs.done = cs.index == cs.length
	}
	return
}

// 拼装message
func (cs *chunkStream) assembleMessage() *message {
	msg := message{
		tid:       cs.mtid,
		length:    cs.length,
		timestamp: cs.timestamp,
		payload:   make([]byte, len(cs.payload)),
	}
	copy(msg.payload, cs.payload)
	return &msg
}

func (cs *chunkStream) complete() bool {
	return cs.done
}

// send message m on msid with chunkSize
func (cs *chunkStream) writeMessage(m Messager, msid, chunkSize uint32) (err error) {
	if cs.payload, err = m.Marshal(); err != nil {
		return
	}
	if len(cs.payload) > math.MaxUint32 {
		return errors.New("message length overflow")
	}
	length := uint32(len(cs.payload))
	if length == 0 {
		return
	}
	tmf, mtid, timestamp := uint8(0), m.Tid(), m.Timestamp()
	// 确定fmt
	switch {
	case !cs.done:
		// send message type 0 for uninitinized chunk stream
		tmf = 0
		cs.length = length
		cs.msid = msid
		cs.mtid = mtid
		cs.timestamp = timestamp
		cs.done = true
	case msid != cs.msid:
		// send message type 0 if chunk stream reused
		Log("chunk stream(%d) has been reused: %d -> %d", cs.id, cs.msid, msid)
		tmf = 0
		cs.length = length
		cs.msid = msid
		cs.mtid = mtid
		cs.timestamp = timestamp
	case length != cs.length || mtid != cs.mtid:
		// send message type 1 message length or message type id
		tmf = 1
		cs.length = length
		cs.mtid = mtid
		cs.delta = timestamp - cs.timestamp
		cs.timestamp = timestamp
	default:
		// send message type 2 set time delta
		tmf = 2
		cs.delta = timestamp - cs.timestamp
		cs.timestamp = timestamp
	}
	return cs.writeChunks(tmf, chunkSize)
}

// send basic header and message header
func (cs *chunkStream) writeMessageHeader(tmf uint8) (err error) {
	if err = writeBasicHeader(cs.rw, tmf, cs.id); err != nil {
		return
	}
	switch tmf {
	case 0:
		err = cs.msgHead.writeFmt0(cs.rw)
	case 1:
		err = cs.msgHead.writeFmt1(cs.rw)
	case 2:
		err = cs.msgHead.writeFmt2(cs.rw)
	}
	return
}

// send message payload in chunks
func (cs *chunkStream) writeChunks(tmf uint8, maxSize uint32) (err error) {
	defer func() {
		cs.index = 0
		cs.payload = nil
	}()

	if cs.length == 0 {
		return
	}
	var l, r uint32
	for l < cs.length {
		if r = l + maxSize; r > cs.length {
			r = cs.length
		}
		// send basic header & message header
		if err = cs.writeMessageHeader(tmf); err != nil {
			return
		}
		// send chunk payload
		if _, err = cs.rw.Write(cs.payload[l:r]); err != nil {
			return
		}
		if tmf != 3 {
			tmf = 3
		}
		l = r
	}
	return cs.rw.Flush()
}

func (cs *chunkStream) discard() {
	cs.index = 0
	cs.payload = nil
}
