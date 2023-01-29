package rtmp

import (
	"bufio"
	"context"
	"errors"
	"io"
	"math"
	"net"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/chenyj/rtmp/encoding/amf0"
	"github.com/chenyj/rtmp/encoding/av"
)

var (
	ErrServerClosed = errors.New("rtmp: Server closed")
	ErrDataMissing  = errors.New("rtmp: Protocol data missing")
	ErrProtocol     = errors.New("rtmp: Protocol error")

	windowAckSize    = uint32(2500000)
	msgChanSize      = 32
	defaultMsid      = uint32(7)
	defaultChunkSize = uint32(4096)
)

/*
rtmp服务流程：
rtmp.ListenAndServe -> server.ListenAndServe -> server.Serve -> go conn.Serve
*/

type ReadWriteFlusher interface {
	io.Reader
	WriteFlusher
}

type WriteFlusher interface {
	io.Writer
	Flush() error
}

//原子的bool类型
type atomicBool int32

func (b *atomicBool) isSet() bool { return atomic.LoadInt32((*int32)(b)) != 0 }
func (b *atomicBool) setTrue()    { atomic.StoreInt32((*int32)(b), 1) }
func (b *atomicBool) setFalse()   { atomic.StoreInt32((*int32)(b), 0) }

// message type id
// message stream id
// chunk stream id
// csid 2 for protocol control message and commands

func newConn(s *Server, nc net.Conn) *conn {
	return &conn{
		server:         s,
		rwc:            nc,
		bufr:           bufio.NewReader(nc),
		bufw:           bufio.NewWriter(nc),
		rChunkSize:     128,
		wChunkSize:     128,
		rChunkStream:   make(map[uint32]chunkReader),
		wChunkStream:   make(map[uint32]chunkWriter),
		peerWindowSize: math.MaxUint32,
	}
}

// message stream id -> message stream
//
// chunk stream id |-> message stream
//                 |-> message stream
// A conn represent a low level conn between network
// and chunks are transimited over it.
type conn struct {
	server         *Server
	rwc            net.Conn // 底层网络连接(TCP)
	bufr           *bufio.Reader
	bufw           *bufio.Writer
	mu             sync.Mutex
	rChunkSize     uint32                 // 读chunk大小
	wChunkSize     uint32                 // 写chunk大小
	rChunkStream   map[uint32]chunkReader // read chunk stream
	wChunkStream   map[uint32]chunkWriter // write chunk stream
	wSequence      uint32
	rSequence      uint32
	lastRSequence  uint32
	windowSize     uint32 // 窗口大小
	peerWindowSize uint32 // 对方窗口大小
	app            string
	streamPath     string
	enDumpCmd      bool
	werr           error
}

// +--------------+----------------+--------------------+--------------+
// | Basic Header | Message Header | Extended Timestamp |  Chunk Data  |
// +--------------+----------------+--------------------+--------------+
// |<------------------- Chunk Header ----------------->|
//
// chunk -> message -> handler
func (c *conn) serve() {
	defer c.close()

	// rtmp handshake
	// err := c.handshake()
	err := handshake(c)
	if err != nil {
		Log("rtmp handshake error: %v", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	// handle message loop
	for msg := range c.readMessage(ctx) {
		if err = c.handleMessage(ctx, msg); err != nil {
			break
		}
	}
	cancel()
	Log("handle message error: %s", err)
}

func (c *conn) readMessage(ctx context.Context) <-chan *message {
	ch := make(chan *message, msgChanSize)
	go func(ctx context.Context, ch chan<- *message) {
		defer close(ch)
		for {
			select {
			default:
				tmf, csid, err := readBasicHeader(c) // read Basic Header
				if err != nil {
					Log("read basic header error: %v", err)
					return
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
					Log("read chunk error: %s", err)
					return
				}
				if !cs.complete() {
					continue //消息未读完，继续读下一个chunk
				}

				msg := cs.assembleMessage()
				cs.reset() // reset for the next message

				// handle message
				switch msg.tid {
				case 1, 2, 3, 5, 6:
					// protocol control message
					if err = c.handleProtocol(msg); err != nil {
						Log("handle protocol error: %v", err)
						return
					}
				default:
					ch <- msg
				}
				// send window acknowledgement message
				if c.rSequence-c.lastRSequence >= c.peerWindowSize {
					c.WriteMessage(AcknowledgementMessage{c.rSequence})
				}
			case <-ctx.Done():
				Log("cancel read message")
				return
			}
		}
	}(ctx, ch)
	return ch
}

func (c *conn) WriteMessage(m Messager) (err error) {
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

func (c *conn) writeMessages(m ...Messager) error {
	for i := range m {
		if err := c.WriteMessage(m[i]); err != nil {
			return err
		}
	}
	return nil
}

// 协议控制消息，同步处理
func (c *conn) handleProtocol(msg *message) (err error) {
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

// 异步处理消息
func (c *conn) handleMessage(ctx context.Context, msg *message) (err error) {
	switch msg.tid {
	default:
		Log("未识别的rtmp消息类型: %d", msg.tid)

	case 4: // User Control Message
		var ucm UserControlMessage
		if err = ucm.Unmarshal(msg.payload); err != nil {
			return
		}
		switch ucm.EventType {
		case STREAM_BEGIN:
			Log("stream begin event: sid(%d)", ucm.Param1)
		case STREAM_EOF:
			Log("stream eof event: sid(%d)", ucm.Param1)
		case STREAM_DRY:
			Log("stream dry event: sid(%d)", ucm.Param1)
		case SET_BUFFER_LENGTH:
			Log("set buffer len event: sid(%d) length(%d)", ucm.Param1, ucm.Param2)
		case STREAM_IS_RECORDED:
			Log("stream is recorded event: sid(%d)", ucm.Param1)
		case PING_REQUEST:
			Log("ping request event: timestamp(%d)", ucm.Param1)
		case PING_RESPONSE:
			Log("ping response event: timestamp(%d)", ucm.Param1)
		}

	case 8: // Audio Message
		p := av.AudioPack(msg.timestamp, msg.payload)
		err = serverHandler{c.server}.OnData(c.app, c.streamPath, p)

	case 9: // Video Message
		p := av.VideoPack(msg.timestamp, msg.payload)
		err = serverHandler{c.server}.OnData(c.app, c.streamPath, p)

	case 15: // Data Message (AMF3)
		Log("Data Message AMF3")

	case 16: // Shared Object Message (AMF3)
		Log("Shared Object Message")

	case 17: // Command Message (AMF3)
		Log("Command Message (AMF3)")

	case 18: // Data Message (AMF0)
		p := av.MetaPack(msg.timestamp, msg.payload)
		err = serverHandler{c.server}.OnData(c.app, c.streamPath, p)

	case 19: // Shared Object Message (AMF0)
		Log("Shared Object Message")

	case 20: // Command Message (AFM0)
		// decode amf payload
		var d amf0.Decoder
		if d, err = amf0.NewDecoder(msg.payload); err != nil {
			return
		}
		var cmdName string // command name
		var transId uint32 // transaction id
		if err = d.Decode(&cmdName, &transId); err != nil {
			return
		}
		// write command payload into file
		if c.enDumpCmd {
			write(msg.payload, cmdName+".bin")
		}
		Log("OnCommand: %s", cmdName)

		switch cmdName {
		default:
			Log("未识别的命令：%s", cmdName)

		case CMD_CONNECT:
			var cc ConnectCommand
			if err = d.Decode(&cc); err != nil {
				return
			}
			// set window acknowledge size
			if err = c.writeMessages(
				SetChunkSizeMessage{defaultChunkSize},
				WindowAcknowledgementSizeMessage{windowAckSize}); err != nil {
				return
			}
			c.app = cc.App
			req := Request{
				TransactionID: transId,
				Command:       cmdName,
				Host:          c.rwc.RemoteAddr().String(),
				App:           cc.App,
			}
			err = serverHandler{c.server}.OnCommand(c, &req)

		case CMD_CALL:
		case CMD_CLOSE:
		case CMD_CREATE_STREAM:
			err = c.WriteMessage(CommandMessage{RSP_RESULT, transId, []any{nil, defaultMsid}})

		case CMD_PLAY:
			uri, ok := d.Skip().GetString()
			if !ok {
				return errors.New("decode amf error")
			}
			u, err := url.Parse(uri)
			if err != nil {
				return err
			}
			req := Request{
				TransactionID: transId,
				Command:       cmdName,
				Host:          c.rwc.RemoteAddr().String(),
				App:           c.app,
				StreamPath:    u.Path,
				Form:          u.Query(),
			}
			err = serverHandler{c.server}.OnCommand(c, &req)

		case CMD_PLAY2:
			Log("play2 command")

		case CMD_DELETE_STREAM:
			streamId, ok := d.Skip().GetUint32()
			if !ok {
				return errors.New("decode amf error")
			}
			Log("deleteStream command: %d", streamId)
			// for k, v := range c.rChunkStream {
			// 	if v.msid == streamId {
			// 		delete(c.rChunkStream, k)
			// 	}
			// }

		case CMD_CLOSE_STREAM:
			Log("closeStream command")
		case CMD_RECEIVE_AUDIO:
			Log("receiveAudio command")
		case CMD_RECEIVE_VIDEO:
			Log("receiveVideo command")

		case CMD_PUBLISH:
			var streamName, streamType string
			if err = d.Decode(nil, &streamName, &streamType); err != nil {
				return
			}
			var u *url.URL
			u, err = url.Parse(streamName)
			if err != nil {
				return err
			}
			c.streamPath = u.Path
			// send stream begin
			if err = c.WriteMessage(UserControlMessage{STREAM_BEGIN, defaultMsid, 0}); err != nil {
				return
			}
			req := Request{
				TransactionID: transId,
				Command:       cmdName,
				Host:          c.rwc.RemoteAddr().String(),
				App:           c.app,
				StreamType:    streamType,
				StreamPath:    u.Path,
				Form:          u.Query(),
			}
			err = serverHandler{c.server}.OnCommand(c, &req)

		case CMD_SEEK:
			Log("seek command")
		case CMD_PAUSE:
			Log("pause command")

		case CMD_FCPUBLISH:
			// commandName,TransacationId,object,streamName
			streamName, _ := d.Skip().GetString()
			Log("FCpublish stream: %s", streamName)

		case CMD_FCUNPUBLISH:
			streamName, ok := d.Skip().GetString()
			if !ok {
				return errors.New("decode amf error")
			}
			u, err := url.Parse(streamName)
			if err != nil {
				return err
			}
			req := Request{
				TransactionID: transId,
				Command:       cmdName,
				Host:          c.rwc.RemoteAddr().String(),
				App:           c.app,
				StreamPath:    u.Path,
				Form:          u.Query(),
			}
			err = serverHandler{c.server}.OnCommand(c, &req)

		case CMD_RELEASE_STREAM:
			// commandName,TransacationId,object,streamName
			streamName, _ := d.Skip().GetString()
			Log("releaseStream: %s", streamName)

		case CMD_GET_STREAM_LENGTH:
			streamName, _ := d.Skip().GetString()
			Log("getStreamLength: %s", streamName)
		}

	case 22: // Aggregate Message
		Log("Aggregate Message")
	}
	return
}

func ResponseConnect(w MessageWriter, status bool, desc string) error {
	var info respInfo
	if status {
		info.Level = LVL_STATUS
		info.Code = "NetConnection.Connect.Success"
		info.Desc = "Connection succeeded"
	} else {
		info.Level = LVL_ERROR
		info.Code = "NetConnection.Connect.Refused"
		info.Desc = desc
	}
	return w.WriteMessage(CommandMessage{RSP_RESULT, 1, []any{RespProp, info}})
}

func ResponsePublish(w MessageWriter, status bool, desc string) (err error) {
	var info respInfo
	if status {
		info.Level = LVL_STATUS
		info.Code = "NetStream.Publish.Start"
		info.Desc = "Start publishing"
	} else {
		info.Level = LVL_ERROR
		info.Code = "NetStream.Publish.Error"
		info.Desc = desc
	}
	return w.WriteMessage(CommandMessage{RSP_ON_STATUS, 0, []any{nil, info}})
}

// The server sends an onStatus command messages NetStream.Play.Start & NetStream.Play.Reset
// if the play command sent by the client is successful. NetStream.Play.Reset is sent by the
// server only if the play command sent by the client has set the reset flag. If the stream
// to be played is not found, the Server sends the onStatus message NetStream.Play.StreamNotFound.
func ResponsePlay(w MessageWriter, status bool, desc string) error {
	var info respInfo
	if status {
		info.Level = LVL_STATUS
		info.Code = "NetStream.Play.Start"
		info.Desc = "Start playing"
	} else {
		info.Level = LVL_ERROR
		info.Code = "NetStream.Play.StreamNotFound"
		info.Desc = desc
	}
	return w.WriteMessage(CommandMessage{RSP_ON_STATUS, 0, []any{nil, info}})
}

// implement io.Reader
func (c *conn) Read(p []byte) (n int, err error) {
	n, err = io.ReadFull(c.bufr, p)
	atomic.AddUint32(&c.rSequence, uint32(n))
	return
}

// implement io.Writer
func (c *conn) Write(p []byte) (n int, err error) {
	n, err = c.bufw.Write(p)
	atomic.AddUint32(&c.wSequence, uint32(n))
	return
}

func (c *conn) Flush() error {
	return c.bufw.Flush()
}

func (c *conn) close() {
	Log("close rtmp connection")
	err := c.rwc.Close()
	if err != nil {
		Log("close conn error: %v", err)
	}
}

func getCsidAndMsid(mtid uint8) (csid, msid uint32) {
	switch mtid {
	default:
		return 7, defaultMsid
	case 1, 2, 3, 4, 5, 6: //protocol & user control message
		return 2, 0
	case 8: // audio message
		return 8, defaultMsid
	case 9: // video message
		return 9, defaultMsid
	case 15, 18: // data message
		return 4, defaultMsid
	case 16, 19: // share object message
		return 5, defaultMsid
	case 17, 20: // command message
		return 3, defaultMsid
	case 22: // aggregate message
		return 6, defaultMsid
	}
}

// A Server represent a rtmp server
type Server struct {
	Addr         string
	MaxChunkSize int
	Logger       Logger
	inShutdown   atomicBool
	lock         sync.Mutex
	doneChan     chan struct{}
	onShutdown   []func()
	Handler      Handler
	streams      map[string]Streamer
}

func (s *Server) ListenAndServe() error {
	if s.shuttingDown() {
		return ErrServerClosed
	}
	var addr string
	if addr = s.Addr; addr == "" {
		addr = ":1935"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	printBanner(addr)
	return s.Serve(ln)
}

func (s *Server) Serve(l net.Listener) error {
	var once sync.Once
	defer once.Do(func() { l.Close() })

	for {
		rw, err := l.Accept()
		if err != nil {
			return err
		}
		go newConn(s, rw).serve()
	}
}

func (s *Server) shuttingDown() bool {
	return s.inShutdown.isSet()
}

func ListenAndServe(addr string, handler Handler) error {
	srv := &Server{Addr: addr, Handler: handler}
	return srv.ListenAndServe()
}

type Request struct {
	TransactionID uint32
	Command       string
	Host          string
	App           string
	StreamPath    string
	StreamType    string
	Form          url.Values
}

type MessageWriter interface {
	WriteMessage(Messager) error
}

type Handler interface {
	OnCommand(MessageWriter, *Request) error
	OnData(string, string, *av.Packet) error
}

type handlerFunc func(MessageWriter, *Request) error

type serverHandler struct {
	srv *Server
}

func (sh serverHandler) OnCommand(w MessageWriter, r *Request) error {
	handler := sh.srv.Handler
	if handler == nil {
		handler = DefaultServeMux
	}
	return handler.OnCommand(w, r)
}

func (sh serverHandler) OnData(app, path string, p *av.Packet) error {
	handler := sh.srv.Handler
	if handler == nil {
		handler = DefaultServeMux
	}
	return handler.OnData(app, path, p)
}

var defaultServeMux ServeMux
var DefaultServeMux = &defaultServeMux

type ServeMux struct {
	mux    map[string]handlerFunc
	onData func(string, string, *av.Packet) error
}

func (sm *ServeMux) OnCommand(w MessageWriter, r *Request) error {
	fn, ok := sm.mux[r.Command]
	if !ok {
		switch r.Command {
		default:
			return errors.New("command handler not found")
		case CMD_CONNECT:
			return ResponseConnect(w, true, "")
		case CMD_PUBLISH:
			return ResponsePublish(w, true, "")
		}
	}
	return fn(w, r)
}

func (sm *ServeMux) OnData(app, path string, p *av.Packet) error {
	if sm.onData == nil {
		return nil
	}
	return sm.onData(app, path, p)
}

func HandleCommand(command string, handler func(MessageWriter, *Request) error) {
	if defaultServeMux.mux == nil {
		defaultServeMux.mux = make(map[string]handlerFunc)
	}
	defaultServeMux.mux[command] = handler
}

func HandleData(handler func(string, string, *av.Packet) error) {
	defaultServeMux.onData = handler
}
