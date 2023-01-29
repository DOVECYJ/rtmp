package rtmp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"sync"
)

// TODO: 使用atomic
type scheme bool

func (s scheme) String() string {
	if s {
		return "scheme_1"
	} else {
		return "scheme_0"
	}
}

const (
	SCHEME0 scheme = false
	SCHEME1 scheme = true
)

var (
	_GENUINE_FMS_KEY_ = []byte{
		0x47, 0x65, 0x6e, 0x75, 0x69, 0x6e, 0x65, 0x20,
		0x41, 0x64, 0x6f, 0x62, 0x65, 0x20, 0x46, 0x6c,
		0x61, 0x73, 0x68, 0x20, 0x4d, 0x65, 0x64, 0x69,
		0x61, 0x20, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72,
		0x20, 0x30, 0x30, 0x31, // Genuine Adobe Flash Media Server 001
		0xf0, 0xee, 0xc2, 0x4a, 0x80, 0x68, 0xbe, 0xe8,
		0x2e, 0x00, 0xd0, 0xd1, 0x02, 0x9e, 0x7e, 0x57,
		0x6e, 0xec, 0x5d, 0x2d, 0x29, 0x80, 0x6f, 0xab,
		0x93, 0xb8, 0xe6, 0x36, 0xcf, 0xeb, 0x31, 0xae,
	}
	_GENUINE_FP_KEY_ = []byte{
		0x47, 0x65, 0x6E, 0x75, 0x69, 0x6E, 0x65, 0x20,
		0x41, 0x64, 0x6F, 0x62, 0x65, 0x20, 0x46, 0x6C,
		0x61, 0x73, 0x68, 0x20, 0x50, 0x6C, 0x61, 0x79,
		0x65, 0x72, 0x20, 0x30, 0x30, 0x31, // Genuine Adobe Flash Player 001
		0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8,
		0x2E, 0x00, 0xD0, 0xD1, 0x02, 0x9E, 0x7E, 0x57,
		0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
		0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}
	// 用来验证C1
	fpHashPool = sync.Pool{
		New: func() any {
			return hmac.New(sha256.New, _GENUINE_FP_KEY_[:30])
		},
	}
	// 用来加密S1
	fmsHashPool = sync.Pool{
		New: func() any {
			return hmac.New(sha256.New, _GENUINE_FMS_KEY_[:36])
		},
	}
	// 用来加密S2
	fmsFullHashPool = sync.Pool{
		New: func() any {
			return hmac.New(sha256.New, _GENUINE_FMS_KEY_)
		},
	}
	defaultScheme = SCHEME0
)

// rtmp握手
func handshake(rw ReadWriteFlusher) (err error) {
	c0c1 := make([]byte, 1537)
	// read C0,C1
	if n, err := rw.Read(c0c1); err != nil {
		return err
	} else if n != 1537 {
		return fmt.Errorf("rtmp handshake data miss(%d)", n)
	}
	// 验证rtmp版本
	if c0c1[0] != 3 {
		return fmt.Errorf("unsupport rtmp version: %d", c0c1[0])
	}

	zero := binary.BigEndian.Uint32(c0c1[5:9])
	if zero == 0 {
		err = simpleHandshake(c0c1, rw) // 简单握手
	} else {
		err = complexHandshake(c0c1, rw) // 复杂握手
	}
	if err != nil {
		return
	}
	c2 := c0c1[:1536]
	// read C2
	_, err = rw.Read(c2)
	return
}

// 简单握手
func simpleHandshake(c0c1 []byte, w WriteFlusher) error {
	// write S0 and S1
	n, err := w.Write(c0c1)
	if err != nil {
		return err
	} else if n != len(c0c1) {
		return fmt.Errorf("write part of s0 and s1: %d", n)
	}
	// write S2
	n, err = w.Write(c0c1[1:])
	if err != nil {
		return err
	} else if n != 1536 {
		return fmt.Errorf("write part of s2: %d", n)
	}
	return w.Flush()
}

// 复杂握手
func complexHandshake(c0c1 []byte, w WriteFlusher) error {
	// write S0
	w.Write(c0c1[:1])
	// write S1 and S2
	c1 := c0c1[1:]
	start, digest, err := getSchemeAndDigest(c1)
	if err != nil {
		return err
	}
	// 获取用来加密S2的key
	key, err := hmac_sha256(&fmsFullHashPool, digest)
	if err != nil {
		return err
	}
	// 生成S1
	if digest, err = hmac_sha256(&fmsHashPool, c1[:start], c1[start+32:]); err != nil {
		return err
	}
	copy(c1[start:start+32], digest)
	// write S1
	if _, err = w.Write(c1); err != nil {
		return err
	}
	// 生成S2
	if digest, err = hmacsha256(c1[:1504], key); err != nil {
		return err
	}
	copy(c1[1504:], digest)
	// write S2
	if _, err = w.Write(c1); err != nil {
		return err
	}
	return w.Flush()
}

// 校验C1并获取Digest
// scheme0: time(4) version(4) key(764) digest(764)
// scheme1: time(4) version(4) digest(764) key(764)
// key    : random(offset) key(128) random(632-offset) offset(4)
// digest : offset(4) random(offset) digest(32) random(760-offset)
func getSchemeAndDigest(c1 []byte) (start int, digest []byte, err error) {
	shm := defaultScheme
	start = getDigestStart(c1, shm)
	digest = c1[start : start+32]
	var temp []byte
	temp, err = hmac_sha256(&fpHashPool, c1[:start], c1[start+32:])
	if err != nil || bytes.Equal(digest, temp) {
		return
	}

	// 换一种scheme
	shm = !shm
	start = getDigestStart(c1, shm)
	digest = c1[start : start+32]
	temp, err = hmac_sha256(&fpHashPool, c1[:start], c1[start+32:])
	if err == nil {
		if bytes.Equal(digest, temp) {
			defaultScheme = shm
		} else {
			err = errors.New("C1校验错误")
		}
	}
	return
}

// 获取Digest的起始下标
func getDigestStart(c1 []byte, shm scheme) int {
	var offset []byte
	if shm {
		offset = c1[8:12] // scheme 1
	} else {
		offset = c1[772:776] // scheme 0
	}
	off := int(offset[0]) + int(offset[1]) + int(offset[2]) + int(offset[3])
	off %= 728 // 防止溢出
	if shm {
		return off + 12 // scheme 1
	}
	return off + 776 // scheme 0
}

// 使用key加密data
func hmacsha256(data, key []byte) ([]byte, error) {
	h := hmac.New(sha256.New, key)
	if _, err := h.Write(data); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func hmac_sha256(pool *sync.Pool, data ...[]byte) ([]byte, error) {
	h := pool.Get().(hash.Hash)
	defer func() {
		h.Reset()
		pool.Put(h)
	}()
	for i := range data {
		if _, err := h.Write(data[i]); err != nil {
			return nil, err
		}
	}
	return h.Sum(nil), nil
}
