package rtmp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/chenyj/rtmp/encoding/amf0"
	"github.com/chenyj/rtmp/encoding/av"
)

// message type
const (
	_                           = iota
	SET_CHUNK_SIZE                         //1
	ABORT                                  //2
	ACKNOWLEDGEMENT                        //3
	USER_CONTROL                           //4
	WINDOW_ACKNOWLEDGEMENT_SIZE            //5
	SET_PEER_BANDWIDTH                     //6
	AUDIO                       = iota + 1 //8
	VIDEO                                  //9
	DATA_AMF3                   = iota + 6 //15
	SHARED_OBJECT_AMF3                     //16
	COMMAND_AMF3                           //17
	DATA_AMF0                              //18
	SHARED_OBJECT_AMF0                     //19
	COMMAND_AMF0                           //20
	AGGREGATE                   = iota + 7 //22
)

// user control message's event type
const (
	STREAM_BEGIN       = iota     //0
	STREAM_EOF                    //1
	STREAM_DRY                    //2
	SET_BUFFER_LENGTH             //3
	STREAM_IS_RECORDED            //4
	PING_REQUEST       = iota + 2 //6
	PING_RESPONSE                 //7
)

// rtmp command
const (
	RSP_RESULT            = "_result"
	RSP_ERROR             = "error"
	RSP_ON_STATUS         = "onStatus"
	CMD_CONNECT           = "connect"
	CMD_CALL              = "call"
	CMD_CLOSE             = "close"
	CMD_CREATE_STREAM     = "createStream"
	CMD_PLAY              = "play"
	CMD_PLAY2             = "play2"
	CMD_DELETE_STREAM     = "deleteStream"
	CMD_CLOSE_STREAM      = "closeStream"
	CMD_RECEIVE_AUDIO     = "receiveAudio"
	CMD_RECEIVE_VIDEO     = "receiveVideo"
	CMD_PUBLISH           = "publish"
	CMD_SEEK              = "seek"
	CMD_PAUSE             = "pause"
	CMD_FCPUBLISH         = "FCPublish"
	CMD_FCUNPUBLISH       = "FCUnpublish"
	CMD_RELEASE_STREAM    = "releaseStream"
	CMD_GET_STREAM_LENGTH = "getStreamLength"

	LVL_STATUS  = level("status")
	LVL_WARNING = level("warning")
	LVL_ERROR   = level("error")
)

var (
	MsgLengthErr = errors.New("message is too long")
)

type level string

type Messager interface {
	Tid() uint8
	Timestamp() uint32
	Marshal() ([]byte, error)
}

func NewMessage(p *av.Packet) message {
	return message{
		tid:       p.Type,
		timestamp: p.Timestamp,
		payload:   p.Payload,
	}
}

//  0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |  Message Type |            Payload length (3 bytes)           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                       Timestamp (4 bytes)                     |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |            Stream ID (3 bytes)                |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type MessageHeader struct {
	Type      uint8
	Length    uint32 // 0xFFFFFF(max)
	Timestamp uint32
	StreamID  uint32 // 0xFFFFFF(max)
}

type Message struct {
	Header  MessageHeader
	Payload []byte
}

func (m Message) Tid() uint8 {
	return m.Header.Type
}

func (m Message) Timestamp() uint32 {
	return m.Header.Timestamp
}

func (m Message) Marshal() ([]byte, error) {
	if len(m.Payload) > math.MaxUint32 {
		return nil, MsgLengthErr
	}
	return m.Payload, nil
}

type message struct {
	tid       uint8  // Message Type (Type Id)
	length    uint32 // Payload length
	timestamp uint32 // Timestamp
	payload   []byte // 消息负载
}

func (m message) Tid() uint8 {
	return m.tid
}

func (m message) Timestamp() uint32 {
	return m.timestamp
}

func (m message) Marshal() ([]byte, error) {
	if len(m.payload) > math.MaxUint32 {
		return nil, MsgLengthErr
	}
	return m.payload, nil
}

func (m message) String() string {
	return fmt.Sprintf("message(%d) timestamp(%d) payload(%d)", m.tid, m.timestamp, len(m.payload))
}

type SetChunkSizeMessage struct {
	chunkSize uint32
}

func (m SetChunkSizeMessage) Tid() uint8 {
	return SET_CHUNK_SIZE
}

func (m SetChunkSizeMessage) Timestamp() uint32 {
	return 0
}

func (m SetChunkSizeMessage) Marshal() ([]byte, error) {
	if m.chunkSize < 128 {
		return nil, errors.New("chunk size less than 128")
	}
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, m.chunkSize&0x7FFFFFFF)
	return bs, nil
}

func (m *SetChunkSizeMessage) Unmarshal(bs []byte) error {
	if len(bs) != 4 {
		return ErrDataMissing
	}
	m.chunkSize = binary.BigEndian.Uint32(bs)
	return nil
}

type AbortMessage struct {
	chunkStreamId uint32
}

func (m AbortMessage) Tid() uint8 {
	return ABORT
}

func (m AbortMessage) Timestamp() uint32 {
	return 0
}

func (m AbortMessage) Marshal() ([]byte, error) {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, m.chunkStreamId)
	return bs, nil
}

func (m *AbortMessage) Unmarshal(bs []byte) error {
	if len(bs) != 4 {
		return ErrDataMissing
	}
	m.chunkStreamId = binary.BigEndian.Uint32(bs)
	return nil
}

type AcknowledgementMessage struct {
	sequenceNumber uint32
}

func (m AcknowledgementMessage) Tid() uint8 {
	return ACKNOWLEDGEMENT
}

func (m AcknowledgementMessage) Timestamp() uint32 {
	return 0
}

func (m AcknowledgementMessage) Marshal() ([]byte, error) {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, m.sequenceNumber)
	return bs, nil
}

func (m *AcknowledgementMessage) Unmarshal(bs []byte) error {
	if len(bs) != 4 {
		return ErrDataMissing
	}
	m.sequenceNumber = binary.BigEndian.Uint32(bs)
	return nil
}

type WindowAcknowledgementSizeMessage struct {
	windowSize uint32
}

func (m WindowAcknowledgementSizeMessage) Tid() uint8 {
	return WINDOW_ACKNOWLEDGEMENT_SIZE
}

func (m WindowAcknowledgementSizeMessage) Timestamp() uint32 {
	return 0
}

func (m WindowAcknowledgementSizeMessage) Marshal() ([]byte, error) {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, m.windowSize)
	return bs, nil
}

func (m *WindowAcknowledgementSizeMessage) Unmarshal(bs []byte) error {
	if len(bs) != 4 {
		return ErrDataMissing
	}
	m.windowSize = binary.BigEndian.Uint32(bs)
	return nil
}

type SetPeerBandwidthMesage struct {
	windowSize uint32
	limitType  uint8
}

func (m SetPeerBandwidthMesage) Tid() uint8 {
	return SET_PEER_BANDWIDTH
}

func (m SetPeerBandwidthMesage) Timestamp() uint32 {
	return 0
}

func (m SetPeerBandwidthMesage) Marshal() ([]byte, error) {
	bs := make([]byte, 5)
	binary.BigEndian.PutUint32(bs, m.windowSize)
	bs[4] = m.limitType
	return bs, nil
}

func (m *SetPeerBandwidthMesage) Unmarshal(bs []byte) error {
	if len(bs) != 5 {
		return ErrDataMissing
	}
	m.windowSize = binary.BigEndian.Uint32(bs)
	m.limitType = bs[4]
	return nil
}

// +------------------------------+------------------------
// |     Event Type (16 bits)     |     Event Data
// +------------------------------+------------------------
// User Control Message必须在一个Chunk内发送，也就是说，
// maxChunkSize必须比Event Data大
type UserControlMessage struct {
	EventType uint16
	Param1    uint32
	Param2    uint32
}

func (m UserControlMessage) Tid() uint8 {
	return USER_CONTROL
}

func (m UserControlMessage) Timestamp() uint32 {
	return 0
}

func (m UserControlMessage) Marshal() ([]byte, error) {
	bs := make([]byte, 6, 10)
	binary.BigEndian.PutUint16(bs[0:2], m.EventType)
	binary.BigEndian.PutUint32(bs[2:6], m.Param1)
	if m.EventType == SET_BUFFER_LENGTH {
		bs = bs[:10]
		binary.BigEndian.PutUint32(bs[6:10], m.Param2)
	}
	return bs, nil
}

func (m *UserControlMessage) Unmarshal(bs []byte) error {
	size := len(bs)
	if size < 6 {
		return ErrDataMissing
	}
	m.EventType = binary.BigEndian.Uint16(bs[:2])
	m.Param1 = binary.BigEndian.Uint32(bs[2:6])
	if m.EventType == SET_BUFFER_LENGTH {
		if size != 10 {
			return ErrDataMissing
		}
		m.Param2 = binary.BigEndian.Uint32(bs[6:10])
	}
	return nil
}

// command message
type CommandMessage struct {
	Name          string
	TransactionId uint32
	arr           []any
}

func (m CommandMessage) Tid() uint8 {
	return COMMAND_AMF0
}

func (m CommandMessage) Timestamp() uint32 {
	return 0
}

func (m CommandMessage) Marshal() ([]byte, error) {
	param := append([]any{m.Name, m.TransactionId}, m.arr...)
	return amf0.Encode(param...)
}

func (m *CommandMessage) Unmarshal(bs []byte) error {
	return amf0.Unmarshal(bs, m)
}

var (
	RespProp = respProp{"FMS/3,0,1,123", 15}
)

type respProp struct {
	FmsVer       string `amf:"fmsVer"`
	Capabilities int    `amf:"capabilities"`
}

type respInfo struct {
	Level level  `amf:"level"` // warning,status,error
	Code  string `amf:"code"`
	Desc  string `amf:"description"`
}

type ConnectCommand struct {
	App      string
	Flashver string
	SwfUrl   string
	TcUrl    string
}
