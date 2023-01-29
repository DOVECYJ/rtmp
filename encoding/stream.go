package encoding

import (
	"fmt"
	"io"
	"reflect"
	"unsafe"
)

type Stream interface {
	Raw() []byte       // 获取原始字节切片
	HasMore() bool     // 判断是否读取结束
	Remain() int       // 获取剩余未读取字节数
	Skip(n int) Stream // 跳过n字节

	F64(n *float64) Stream // 读取8字节浮点数
	U64(n *uint64) Stream  // 读取8字节uint64
	I64(n *int64) Stream

	F32(n *float32) Stream // 读取4字节浮点数
	U32(n *uint32) Stream  // 读取4字节uint32
	I32(n *int32) Stream   // 读取4字节int32

	U24(n *uint32) Stream // 读取3字节uint32
	I24(n *int32) Stream

	U16(n *uint16) Stream // 读取2字节uint16
	I16(n *int16) Stream

	U8(n *uint8) Stream // 读取1字节u8
	I8(n *int8) Stream

	Bool(n *bool) Stream // 读取一字节bool
	Byte(b *byte) Stream // 读取1个字节

	Bytes(bs []byte) Stream // 读取字节切片

	Produce(n int) (Stream, error)

	ByteAt(offset int) (byte, error) // 获取offset处字节
	Slice(n int) ([]byte, error)
	String(n int) (string, error)
	ReadAll() []byte

	Error() error // 获取error
	Debug()
}

type ByteStream struct {
	index int
	raw   []byte
	Err   error
}

func (s *ByteStream) Raw() []byte {
	return s.raw
}

// 判断读取是否结束
func (s *ByteStream) HasMore() bool {
	return s.index < len(s.raw)
}

// 跳过n字节
func (s *ByteStream) Skip(n int) Stream {
	s.index += n
	return s
}

// 读取8字节浮点数
func (s *ByteStream) F64(n *float64) Stream {
	if !s.checkBeforeRead(8) {
		return s
	}
	p := (*[8]byte)(unsafe.Pointer(n))
	(*p)[7] = s.raw[s.index]
	(*p)[6] = s.raw[s.index+1]
	(*p)[5] = s.raw[s.index+2]
	(*p)[4] = s.raw[s.index+3]
	(*p)[3] = s.raw[s.index+4]
	(*p)[2] = s.raw[s.index+5]
	(*p)[1] = s.raw[s.index+6]
	(*p)[0] = s.raw[s.index+7]
	s.index += 8
	return s
}

func (s *ByteStream) U64(n *uint64) Stream {
	if !s.checkBeforeRead(8) {
		return s
	}
	p := (*[8]byte)(unsafe.Pointer(n))
	(*p)[7] = s.raw[s.index]
	(*p)[6] = s.raw[s.index+1]
	(*p)[5] = s.raw[s.index+2]
	(*p)[4] = s.raw[s.index+3]
	(*p)[3] = s.raw[s.index+4]
	(*p)[2] = s.raw[s.index+5]
	(*p)[1] = s.raw[s.index+6]
	(*p)[0] = s.raw[s.index+7]
	s.index += 8
	return s
}
func (s *ByteStream) I64(n *int64) Stream {
	if !s.checkBeforeRead(8) {
		return s
	}
	p := (*[8]byte)(unsafe.Pointer(n))
	(*p)[7] = s.raw[s.index]
	(*p)[6] = s.raw[s.index+1]
	(*p)[5] = s.raw[s.index+2]
	(*p)[4] = s.raw[s.index+3]
	(*p)[3] = s.raw[s.index+4]
	(*p)[2] = s.raw[s.index+5]
	(*p)[1] = s.raw[s.index+6]
	(*p)[0] = s.raw[s.index+7]
	s.index += 8
	return s
}
func (s *ByteStream) F32(n *float32) Stream {
	if !s.checkBeforeRead(4) {
		return s
	}
	p := (*[4]byte)(unsafe.Pointer(n))
	(*p)[3] = s.raw[s.index]
	(*p)[2] = s.raw[s.index+1]
	(*p)[1] = s.raw[s.index+2]
	(*p)[0] = s.raw[s.index+3]
	s.index += 4
	return s
}

// 读取4字节uint32
// 此实现为小端序系统
func (s *ByteStream) U32(n *uint32) Stream {
	if !s.checkBeforeRead(4) {
		return s
	}
	p := (*[4]byte)(unsafe.Pointer(n))
	(*p)[3] = s.raw[s.index]
	(*p)[2] = s.raw[s.index+1]
	(*p)[1] = s.raw[s.index+2]
	(*p)[0] = s.raw[s.index+3]
	s.index += 4
	return s
}

// 读取4字节uint32
// 此实现为小端序系统
func (s *ByteStream) I32(n *int32) Stream {
	if !s.checkBeforeRead(4) {
		return s
	}
	p := (*[4]byte)(unsafe.Pointer(n))
	(*p)[3] = s.raw[s.index]
	(*p)[2] = s.raw[s.index+1]
	(*p)[1] = s.raw[s.index+2]
	(*p)[0] = s.raw[s.index+3]
	s.index += 4
	return s
}

// 读取3字节uint32
// 此实现为小端序系统
func (s *ByteStream) U24(n *uint32) Stream {
	if !s.checkBeforeRead(3) {
		return s
	}
	p := (*[4]byte)(unsafe.Pointer(n))
	(*p)[3] = 0
	(*p)[2] = s.raw[s.index]
	(*p)[1] = s.raw[s.index+1]
	(*p)[0] = s.raw[s.index+2]
	s.index += 3
	return s
}
func (s *ByteStream) I24(n *int32) Stream {
	if !s.checkBeforeRead(3) {
		return s
	}
	p := (*[4]byte)(unsafe.Pointer(n))
	(*p)[3] = 0
	(*p)[2] = s.raw[s.index]
	(*p)[1] = s.raw[s.index+1]
	(*p)[0] = s.raw[s.index+2]
	s.index += 3
	return s
}

// 读取2字节uint16
// 此实现为小端序系统
func (s *ByteStream) U16(n *uint16) Stream {
	if !s.checkBeforeRead(2) {
		return s
	}
	p := (*[2]byte)(unsafe.Pointer(n))
	(*p)[1] = s.raw[s.index]
	(*p)[0] = s.raw[s.index+1]
	s.index += 2
	return s
}
func (s *ByteStream) I16(n *int16) Stream {
	if !s.checkBeforeRead(2) {
		return s
	}
	p := (*[2]byte)(unsafe.Pointer(n))
	(*p)[1] = s.raw[s.index]
	(*p)[0] = s.raw[s.index+1]
	s.index += 2
	return s
}

// 读取1字节uint8
func (s *ByteStream) U8(n *uint8) Stream {
	if !s.checkBeforeRead(1) {
		return s
	}
	(*n) = uint8(s.raw[s.index])
	s.index++
	return s
}
func (s *ByteStream) I8(n *int8) Stream {
	if !s.checkBeforeRead(1) {
		return s
	}
	(*n) = int8(s.raw[s.index])
	s.index++
	return s
}

func (s *ByteStream) Bool(n *bool) Stream {
	if !s.checkBeforeRead(1) {
		return s
	}
	p := (*byte)(unsafe.Pointer(n))
	(*p) = s.raw[s.index]
	s.index++
	return s
}

// 读取一个字节
func (s *ByteStream) Byte(b *byte) Stream {
	if !s.checkBeforeRead(1) {
		return s
	}
	(*b) = s.raw[s.index]
	s.index++
	return s
}

// 读取字节切片
func (s *ByteStream) Bytes(bs []byte) Stream {
	if !s.checkBeforeRead(len(bs)) {
		return s
	}
	for i := range bs {
		bs[i] = s.raw[s.index+i]
	}
	s.index += len(bs)
	return s
}

func (s *ByteStream) Produce(n int) (Stream, error) {
	if !s.checkBeforeRead(n) {
		return nil, s.Err
	}
	start := s.index
	s.index += n
	return &ByteStream{
		raw: s.raw[start:s.index:s.index],
	}, nil
}

// 获取offset处字节
func (s *ByteStream) ByteAt(offset int) (byte, error) {
	if !s.checkBeforeRead(offset) {
		return 0, s.Err
	}
	return s.raw[s.index+offset], nil
}

func (s *ByteStream) Slice(n int) (bs []byte, err error) {
	if !s.checkBeforeRead(n) {
		err = s.Err
		return
	}
	start := s.index
	s.index += n
	bs = s.raw[start:s.index:s.index]
	return
}
func (s *ByteStream) String(n int) (ss string, err error) {
	if !s.checkBeforeRead(n) {
		err = s.Err
		return
	}
	h := (*reflect.StringHeader)(unsafe.Pointer(&ss))
	h.Data = uintptr(unsafe.Pointer(&s.raw[s.index]))
	h.Len = n
	s.index += n
	return
}

func (s *ByteStream) ReadAll() []byte {
	start := s.index
	s.index = len(s.raw)
	return s.raw[start:s.index:s.index]
}

func (s *ByteStream) Error() error {
	return s.Err
}

func (s *ByteStream) Debug() {
	fmt.Println("index:", s.index)
}

// 剩余字节数
func (s *ByteStream) Remain() int {
	return len(s.raw) - s.index
}

// 检查是否可以读取
func (s *ByteStream) checkBeforeRead(n int) bool {
	if s.Err != nil {
		return false
	}
	if len(s.raw) < s.index+n {
		s.Err = fmt.Errorf("no more %d bytes for read", n)
		return false
	}
	return true
}

func NewByteStream(bs []byte) Stream {
	stream := &ByteStream{
		raw: bs,
	}
	if len(bs) == 0 {
		stream.Err = io.EOF
	}
	return stream
}
