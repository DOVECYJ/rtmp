package rtmp

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/chenyj/rtmp/encoding/av"
)

var cacheFrameSize = 3000

var streamPool = sync.Pool{
	New: func() any {
		return NewStream(cacheFrameSize)
	},
}

func GetStream() Streamer {
	return streamPool.Get().(Streamer)
}

func ReleaseStream(s Streamer) {
	streamPool.Put(s)
}

type Streamer interface {
	Write(*av.Packet)
	GetConfigFrame() []*av.Packet
	Iterator() Iterator
	IsPublishing() bool
	Publish()
	Unpublish()
	Subscribs() int32
}

type Iterator interface {
	Next() (*av.Packet, error)
	Do(context.Context, func(*av.Packet) error) error
	Release()
}

func NewStream(size int) Streamer {
	var s avStream
	s.ring = New(size)
	s.size = size
	s.Add(3)
	s.ring.Add(1)
	return &s
}

// A stream is a infinity sequence.
type avStream struct {
	sync.WaitGroup            // 读配置帧的锁
	meta           *av.Packet // meta data
	audio0         *av.Packet // audio config
	video0         *av.Packet // video config
	ring           *Ring      // 数据包队列
	entry          *Ring      // last key frame
	size           int        // 队列大小
	sequence       uint64     // 数据包编号
	onlyAudio      bool       // 是否只存储音频
	isPublishing   atomicBool // 是否在发布
	subscriber     int32      // 订阅者数量
}

// Write put a Packet to the stream sequence.
func (s *avStream) Write(p *av.Packet) {
	if p == nil {
		s.ring.Packet = nil
		s.ring.sequence = s.sequence
		s.ring = s.ring.Next()
		s.sequence++
		s.ring.Prev().Done()
		return
	}

	switch {
	case s.meta == nil && p.IsMeta():
		s.meta = p
		s.Done()
		return
	case s.audio0 == nil && p.IsAudio() && p.IsConfig:
		s.audio0 = p
		s.Done()
		return
	case s.video0 == nil && p.IsVideo() && p.IsConfig:
		s.video0 = p
		s.Done()
		return
	}

	// 普通数据帧或重发的meta
	// 写入数据帧
	s.ring.Packet = p
	s.ring.sequence = s.sequence
	if p.IsKeyFrame {
		s.entry = s.ring
	}
	s.ring = s.ring.NextW()
	s.sequence++
	s.ring.Prev().Done()
}

func (s *avStream) GetConfigFrame() []*av.Packet {
	s.Wait()
	packets := []*av.Packet{s.meta}
	if !s.onlyAudio {
		packets = append(packets, s.video0)
	}
	packets = append(packets, s.audio0)
	return packets
}

func (s *avStream) IsPublishing() bool {
	return s.isPublishing.isSet()
}

func (s *avStream) Publish() {
	s.isPublishing.setTrue()
}

func (s *avStream) Unpublish() {
	s.isPublishing.setFalse()
}

func (s *avStream) Subscribs() int32 {
	return atomic.LoadInt32(&s.subscriber)
}

func (s *avStream) Iterator() Iterator {
	atomic.AddInt32(&s.subscriber, 1)
	return &iterator{s: s}
}

// 流迭代器
type iterator struct {
	s        *avStream
	r        *Ring
	sequence uint64
	status   uint8
	// need mutex to ensure concurrent safe
}

func (i *iterator) Next() (p *av.Packet, err error) {
	if i.s == nil {
		return nil, errors.New("invalid iterator on nil stream")
	}

	switch i.status {
	case 0:
		i.s.Wait()
		i.status |= 1
		return i.s.meta, nil
	case 1:
		i.s.Wait()
		i.status |= 2
		return i.s.video0, nil
	case 3:
		i.s.Wait()
		i.status |= 4
		return i.s.audio0, nil
	}

	if i.r == nil {
		i.moveTo(i.s.entry)
	}
	i.r.Wait()

	innerSequence := i.r.sequence
	switch {
	case i.sequence == innerSequence:
		// 正常读取
		if i.r.Packet == nil {
			err = io.EOF
		} else {
			p = i.r.Packet
			i.r = i.r.Next()
			i.sequence++
		}
	case i.sequence < innerSequence:
		// 数据被覆盖，启动丢帧
		r := i.s.ring
		for r.Packet != nil && !r.IsKeyFrame {
			//TODO: check r.Value.(packet).Packet != nil
			r = r.Next()
		}
		if r.Packet == nil {
			err = io.EOF
		} else {
			p = r.Packet
			i.moveTo(r.Next())
		}
	case i.sequence > innerSequence:
		// 超过了uint64的上限
		err = errors.New("bad sequence")
	}
	return
}

func (i *iterator) Do(ctx context.Context, fn func(*av.Packet) error) (err error) {
	if i.s == nil {
		return errors.New("invalid iterator on nil stream")
	}
	i.s.Wait()
	if err = fn(i.s.meta); err != nil {
		return
	}
	if !i.s.onlyAudio {
		if err = fn(i.s.video0); err != nil {
			return
		}
	}
	if err = fn(i.s.audio0); err != nil {
		return
	}

	if i.r == nil {
		// find entry to the stream
		i.moveTo(i.s.entry)
	}
	for {
		select {
		default:
			i.r.Wait()
			innerSequence := i.r.sequence
			switch {
			case i.sequence == innerSequence:
				// 正常读取
				if i.r.Packet == nil {
					return io.EOF
				}
				if err = fn(i.r.Packet); err != nil {
					return
				}
				i.next()
			case i.sequence < innerSequence:
				// 数据被覆盖，启动丢帧
				r := i.s.ring
				for r.Packet != nil && !r.IsKeyFrame {
					r = r.Next()
				}
				i.moveTo(r)
			case i.sequence > innerSequence:
				// 超过了uint64的上限
				return errors.New("bad sequence")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (i *iterator) Release() {
	atomic.AddInt32(&i.s.subscriber, -1)
}

func (i *iterator) moveTo(r *Ring) {
	i.r = r
	i.sequence = r.sequence
}

func (i *iterator) next() {
	i.r = i.r.Next()
	i.sequence++
}
