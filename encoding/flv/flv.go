package flv

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/chenyj/rtmp/encoding"
)

type any interface{}

const (
	FLV_TAG_AUDIO = 8
	FLV_TAG_VIDEO = 9
	FLV_TAG_DATA  = 18
)

const (
	CODEC_JPEG = iota + 1
	CODEC_H263
	CODEC_SCREEN_VIDEO
	CODEC_VP6
	CODEC_VP6_ALPHA
	CODEC_SCREEN_VIDEO_V2
	CODEC_AVC
)

const (
	ADPCM = iota + 1
	MP3
	PCM
	NELLYMOSER16
	NELLYMOSER8
	NELLYMOSER
	G711A
	G711MU
	_RESERVED_
	AAC
	SPEEX
	_
	_
	MP3_8KHZ
	DEVICE
)

var (
	_FLV_         = [3]byte{'F', 'L', 'V'}
	FLV_FMT_ERROR = errors.New("flv format error")
)

// flv头部
type FlvHeader struct {
	flv      [3]byte
	Version  uint8
	HasAudio bool
	HasVideo bool
	Size     uint32
}

// flv tag
type FlvTag struct {
	Header FlvTagHeader
	Data   encoding.Stream
}

func (f FlvTag) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("tag header: %+v\ntag body:", f.Header))
	for i, b := range f.Data.Raw() {
		if i&0x0F == 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(fmt.Sprintf("%2X ", b))
	}
	return sb.String()
}

func (f *FlvTag) toFlvScriptTag() (tag FlvScriptTag, err error) {
	// tag.Header = f.Header
	// a := &AMF{s: f.Data}
	// if matedata, err := a.ReadAny(); err != nil {
	// 	return tag, err
	// } else {
	// 	tag.Method = matedata.(string)
	// }
	// if mateData, err := a.ReadAny(); err != nil {
	// 	return tag, err
	// } else {
	// 	tag.MateData = mateData.(map[string]any)
	// }
	return
}

func (f *FlvTag) toFlvVideoTag() (tag FlvVideoTag, err error) {
	tag.Header = f.Header
	var b byte
	if err = f.Data.Byte(&b).Error(); err != nil {
		return
	}
	tag.FrameType = (b & 0xF0) >> 4
	tag.CodecID = b & 0x0F
	tag.Data = f.Data
	return
}

// flv tag头部
type FlvTagHeader struct {
	TagType    uint8
	DataSize   uint32
	Timestamp  uint32
	StreamID   uint32
	PreTagSzie uint32
}

type FlvScriptTag struct {
	Header   FlvTagHeader
	Method   string
	MateData map[string]any
}

type FlvVideo0Tag struct {
	Header    FlvTagHeader
	FrameType uint8
	CodecId   uint8
}

type FlvAudioTag struct {
	Header    FlvTagHeader
	SoundFmat uint8
	SoundRate uint8
	SoundSize uint8
	SoundType uint8
	Data      encoding.Stream
}

type AACPacket struct {
	AACPacketType uint8
	Playload      []byte
}

type AACAudioSpecificConfig struct {
	AudioObjectType      uint8
	SamplingFrequency    uint8
	ChannelConfiguration uint8
	FrameLengthFlag      uint8
	DependsOnCoreCoder   uint8
	ExtensionFlag        uint8
}

type FlvVideoTag struct {
	Header    FlvTagHeader
	FrameType uint8
	CodecID   uint8
	Data      encoding.Stream
}

type AVCPacket struct {
	AVCPacketType         uint8
	CompositionTimeOffset uint32
	Data                  encoding.Stream
}

type AVCDecoderConfigurationRecord struct {
	ConfigurationVersion uint8
	AVCProfileIndication uint8
	ProfileCompatibility uint8
	AVCLevelIndication   uint8
	LengthSizeMinusOne   uint8
	NumOfSPS             uint8
	SPSLength            uint16
	SPS                  []byte
	NumOfPPS             uint8
	PPSLength            uint16
	PPS                  []byte
}

func New(name string) (*FlvReader, error) {
	if name == "" {
		return nil, errors.New("empty flv name")
	}
	bs, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return &FlvReader{encoding.NewByteStream(bs)}, nil
}

type FlvReader struct {
	stream encoding.Stream
}

func (f *FlvReader) HasMore() bool {
	return f.stream.HasMore()
}

// 读取flv头部
func (f *FlvReader) ReadFlvHeader() (h FlvHeader, err error) {
	//FLV
	f.stream.Byte(&h.flv[0]).Byte(&h.flv[1]).Byte(&h.flv[2])
	if h.flv != _FLV_ {
		err = FLV_FMT_ERROR
		return
	}
	var flag byte
	f.stream.Byte(&h.Version).Byte(&flag).U32(&h.Size)
	h.HasAudio = flag&0x04 == 0x04
	h.HasVideo = flag&0x01 == 0x01
	err = f.stream.Error()
	return
}

// 读取flv tag头部
func (f *FlvReader) ReadFlvTagHeader() (h FlvTagHeader, err error) {
	var externTimestamp uint8
	f.stream.U32(&h.PreTagSzie).Byte(&h.TagType).U24(&h.DataSize).U24(&h.Timestamp).U8(&externTimestamp)
	if h.Timestamp == 0xFFFFFF {
		h.Timestamp += uint32(externTimestamp)
	}
	f.stream.U24(&h.StreamID)
	err = f.stream.Error()
	return
}

// 读取一个flv tag
func (f *FlvReader) ReadFlvTag() (tag FlvTag, err error) {
	tag.Header, err = f.ReadFlvTagHeader()
	if err != nil {
		return
	}
	tag.Data, err = f.stream.Produce(int(tag.Header.DataSize))
	if err != nil {
		fmt.Println(err)
	} else {
		//tag.show()
	}
	return
}

func (f *FlvReader) log() {
	f.stream.Debug()
}

func (f *FlvReader) ParseFlvTag() (ch chan any, err error) {
	ch = make(chan any, 1)
	go func() {
		for f.HasMore() {
			tag, err := f.ReadFlvTag()
			if err != nil {
				ch <- err
				close(ch)
				break
			}
			switch tag.Header.TagType {
			case 8: //audio tag
				if at, err := DecodeFlvAudioTag(tag); err == nil {
					ch <- at
				}
			case 9: //video tag
				if vt, err := tag.toFlvVideoTag(); err == nil {
					ch <- vt
				}
				ch <- tag
			case 18: //script tag
				if st, err := DecodeFlvScriptTag(tag); err == nil {
					ch <- st
				}
			default:
				ch <- tag
			}
		}
	}()
	return
}

// 解码音频Tag
func DecodeFlvAudioTag(tag FlvTag) (a FlvAudioTag, err error) {
	a.Header = tag.Header
	var b byte
	if err = tag.Data.Byte(&b).Error(); err != nil {
		return
	}
	a.SoundFmat = b >> 4
	a.SoundRate = b >> 2 & 0x03
	a.SoundSize = b >> 1 & 0x01
	a.SoundType = b & 0x01
	a.Data, err = tag.Data.Produce(tag.Data.Remain())
	return
}

func DecodeFlvScriptTag(tag FlvTag) (s FlvScriptTag, err error) {
	// s.Header = tag.Header
	// a := &AMF{s: tag.Data}
	// if matedata, err := a.ReadAny(); err != nil {
	// 	return s, err
	// } else {
	// 	s.Method = matedata.(string)
	// }
	// if mateData, err := a.ReadAny(); err != nil {
	// 	return s, err
	// } else {
	// 	s.MateData = mateData.(map[string]any)
	// }
	return
}

func DecodeVideoTag0(tag FlvVideoTag) (v FlvVideo0Tag, err error) {
	return
}

func ReadFlv(name string) (encoding.Stream, error) {
	if name == "" {
		return nil, errors.New("empty flv name")
	}
	bs, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return encoding.NewByteStream(bs), nil
}

func DecodeFlvHeader(s encoding.Stream) (h FlvHeader, err error) {
	// 读取FLV
	err = s.Byte(&h.flv[0]).Byte(&h.flv[1]).Byte(&h.flv[2]).Error()
	if h.flv != _FLV_ || err != nil {
		err = FLV_FMT_ERROR
		return
	}
	var flag byte
	// 读取FLV版本、标识符、头部大小
	err = s.U8(&h.Version).Byte(&flag).U32(&h.Size).Error()
	h.HasAudio = flag&0x04 == 0x04
	h.HasVideo = flag&0x01 == 0x01
	return
}

// 读取一个FLV Tag
func DecodeFlvTag(s encoding.Stream) (t FlvTag, err error) {
	var externTimestamp uint8
	err = s.
		U32(&t.Header.PreTagSzie). //前一个Tag大小
		U8(&t.Header.TagType).     //Tag类型
		U24(&t.Header.DataSize).   //Tag Data大小
		U24(&t.Header.Timestamp).  //时间戳
		U8(&externTimestamp).      //扩展时间戳
		U24(&t.Header.StreamID).   //流ID
		Error()
	if err != nil {
		return
	}
	//拼完整时间戳
	if externTimestamp != 0 {
		t.Header.Timestamp |= uint32(externTimestamp) << 24
	}
	//读取Tag Data
	t.Data, err = s.Produce(int(t.Header.DataSize))
	return
}

type FlvDataTag struct {
	Header   FlvTagHeader
	Method   string
	MateData map[string]interface{}
}

// func DecodeFlvDataTag(t FlvTag) (d FlvDataTag, err error) {
// 	d.Header = t.Header
// 	var val interface{}
// 	var ok bool
// 	if val, err = DecodeAMF(t.Data); err != nil {
// 		return
// 	} else if d.Method, ok = val.(string); !ok {
// 		err = FLV_FMT_ERROR
// 		return
// 	}

// 	if val, err = DecodeAMF(t.Data); err != nil {
// 		return
// 	} else if d.MateData, ok = val.(map[string]interface{}); !ok {
// 		err = FLV_FMT_ERROR
// 		return
// 	}
// 	return
// }

// 解码AAC Packet
func DecodeAACPacket(t FlvAudioTag) (p AACPacket, err error) {
	if t.SoundFmat != AAC {
		err = errors.New("not aac format")
		return
	}
	if err = t.Data.U8(&p.AACPacketType).Error(); err != nil {
		return
	}
	p.Playload = t.Data.ReadAll()
	return
}

// 解码AudioSpecificConfig
func DecodeAACAudioSpecificConfig(p AACPacket) (a AACAudioSpecificConfig, err error) {
	if len(p.Playload) != 2 {
		err = FLV_FMT_ERROR
	}
	a.AudioObjectType = p.Playload[0] >> 3
	a.SamplingFrequency = (p.Playload[0] << 1 & 0x0F) | (p.Playload[1] >> 7)
	a.ChannelConfiguration = p.Playload[1] >> 3 & 0x0F
	a.FrameLengthFlag = p.Playload[1] >> 2 & 0x01
	a.DependsOnCoreCoder = p.Playload[1] >> 1 & 0x01
	a.ExtensionFlag = p.Playload[1] & 0x01
	return
}

// 解码FLV视频Tag
func DecodeFlvVideoTag(t FlvTag) (v FlvVideoTag, err error) {
	v.Header = t.Header
	var b byte
	if err = t.Data.Byte(&b).Error(); err != nil {
		return
	}
	v.FrameType = b >> 4 & 0x0F
	v.CodecID = b & 0x0F
	v.Data, err = t.Data.Produce(t.Data.Remain())
	return
}

// 解码AVC包
func DecodeAVCPacket(t FlvVideoTag) (p AVCPacket, err error) {
	if t.CodecID != CODEC_AVC {
		err = errors.New("not avc format")
	}
	err = t.Data.U8(&p.AVCPacketType).
		U24(&p.CompositionTimeOffset).
		Error()
	if err != nil {
		return
	}
	p.Data, err = t.Data.Produce(t.Data.Remain())
	return
}

// 解码AVCDecoderConfigurationRecord
func DecodeAVCDecoderConfigurationRecord(p AVCPacket) (a AVCDecoderConfigurationRecord, err error) {
	err = p.Data.U8(&a.ConfigurationVersion).
		U8(&a.AVCProfileIndication).
		U8(&a.ProfileCompatibility).
		U8(&a.AVCLevelIndication).
		U8(&a.LengthSizeMinusOne).
		U8(&a.NumOfSPS).
		U16(&a.SPSLength).
		Error()
	a.LengthSizeMinusOne &= 0x03
	a.NumOfSPS &= 0x1F
	if err != nil {
		return
	}
	if a.SPS, err = p.Data.Slice(int(a.SPSLength)); err != nil {
		return
	}
	if err = p.Data.U8(&a.NumOfPPS).U16(&a.PPSLength).Error(); err != nil {
		return
	}
	a.PPS, err = p.Data.Slice(int(a.PPSLength))
	return
}
