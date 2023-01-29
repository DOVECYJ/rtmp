package amf0

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"sync"
)

var epool = sync.Pool{New: func() any { return new(encodeState).init() }}

var objEnd = []byte{0x00, 0x00, 0x09}

type field struct {
	index int
	name  string
	value reflect.Value
}

func Marshal(v any) ([]byte, error) {
	e := epool.Get().(*encodeState)
	defer func() {
		epool.Put(e.reset())
	}()
	bs := e.marshal(v)
	return bs, e.err
}

func Encode(v ...any) ([]byte, error) {
	e := epool.Get().(*encodeState)
	defer func() {
		epool.Put(e.reset())
	}()
	bs := e.encode(v...)
	return bs, e.err
}

type encodeState struct {
	buf   bytes.Buffer
	order binary.ByteOrder
	cache []byte
	err   error
}

func (e *encodeState) init() *encodeState {
	e.order = binary.BigEndian
	e.cache = make([]byte, 0, 8)
	return e
}

func (e *encodeState) reset() *encodeState {
	e.err = nil
	e.cache = e.cache[:0]
	e.buf.Reset()
	return e
}

func (e *encodeState) marshal(v any) []byte {
	e.envalue(reflect.ValueOf(v))
	return e.buf.Bytes()
}

func (e *encodeState) encode(v ...any) []byte {
	for _, m := range v {
		e.value(m)
	}
	return e.buf.Bytes()
}

func (e *encodeState) envalue(v reflect.Value) {
	switch v.Kind() {
	default:
		return
	case reflect.Bool:
		e.tf(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.number(float64(v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		e.number(float64(v.Uint()))
	case reflect.Float32, reflect.Float64:
		e.number(v.Float())
	case reflect.String:
		e.str(v.String())
	case reflect.Array, reflect.Slice:
		e.arr(v)
	case reflect.Map:
		if v.IsNil() {
			e.buf.WriteByte(AMF_NULL)
			return
		}
		kt := v.Type().Key()
		if kt.Kind() != reflect.String {
			e.err = fmt.Errorf("amf0 encode map error: can not handle key type: %s", kt)
			return
		}
		e.object(v)
	case reflect.Struct:
		e.enstruct(v)
	case reflect.Interface:
		if v.IsNil() {
			e.buf.WriteByte(AMF_NULL)
		} else {
			e.envalue(v.Elem())
		}
	case reflect.Pointer:
		if v.IsNil() {
			e.buf.WriteByte(AMF_NULL)
		} else {
			e.envalue(v.Elem())
		}
	}
}

func (e *encodeState) value(v any) {
	if v == nil {
		e.buf.WriteByte(AMF_NULL)
		return
	}
	switch v := v.(type) {
	case uint8:
		e.number(float64(v))
	case int8:
		e.number(float64(v))
	case uint16:
		e.number(float64(v))
	case int16:
		e.number(float64(v))
	case uint32:
		e.number(float64(v))
	case int32:
		e.number(float64(v))
	case uint64:
		e.number(float64(v))
	case int64:
		e.number(float64(v))
	case uint:
		e.number(float64(v))
	case int:
		e.number(float64(v))
	case float32:
		e.number(float64(v))
	case float64:
		e.number(v)
	case bool:
		e.tf(v)
	case string:
		e.str(v)
	case map[string]any:
		e.kv(v)
	case []any:
		e.slice(v)
	default:
		rv := reflect.ValueOf(v)
		e.envalue(rv)
	}
}

func (e *encodeState) number(v float64) {
	e.buf.WriteByte(AMF_NUMBER)
	e.cache = e.cache[:8]
	e.order.PutUint64(e.cache, math.Float64bits(v))
	e.buf.Write(e.cache)
}

func (e *encodeState) tf(v bool) {
	e.buf.WriteByte(AMF_BOOLEAN)
	if v {
		e.buf.WriteByte(1)
	} else {
		e.buf.WriteByte(0)
	}
}

func (e *encodeState) str(s string) {
	size := len(s)
	if size > math.MaxUint16 {
		e.longstr(s)
		return
	}
	e.buf.WriteByte(AMF_STRING)
	e.u16(uint16(size))
	e.buf.WriteString(s)
}

func (e *encodeState) longstr(s string) {
	size := len(s)
	if size > math.MaxUint32 {
		e.err = fmt.Errorf("amf0 encode long string error: string len %d is longer than %d.", size, math.MaxUint32)
		return
	}
	e.buf.WriteByte(AMF_LONG_STRING)
	e.u32(uint32(size))
	e.buf.WriteString(s)
}

//TODO: handle nil value
func (e *encodeState) kv(m map[string]any) {
	e.buf.WriteByte(AMF_OBJECT)
	for k, v := range m {
		size := len(k)
		if size > math.MaxUint16 {
			e.err = fmt.Errorf("amf0 encode object error: key len %d is longer than %d.", size, math.MaxUint16)
			return
		}
		e.u16(uint16(size))
		e.buf.WriteString(k)
		e.value(v)
	}
	e.buf.Write(objEnd)
}

//TODO: handle nil value
func (e *encodeState) ecmarr(m map[string]any) {
	e.buf.WriteByte(AMF_ECMA_ARRAY)
	size := len(m)
	if size > math.MaxUint32 {
		e.err = fmt.Errorf("amf0 encode ecma array error: arr len %d is longer than %d.", size, math.MaxUint32)
		return
	}
	e.u32(uint32(size))
	for k, v := range m {
		size = len(k)
		if size > math.MaxUint16 {
			e.err = fmt.Errorf("amf0 encode ecma array error: key len %d is longer than %d.", size, math.MaxUint16)
			return
		}
		e.u16(uint16(size))
		e.buf.WriteString(k)
		e.value(v)
	}
	e.buf.Write(objEnd)
}

func (e *encodeState) slice(v []any) {
	e.buf.WriteByte(AMF_STRICT_ARRAY)
	size := len(v)
	if size > math.MaxUint32 {
		e.err = fmt.Errorf("amf0 encode array error: arr len %d is longer than %d.", size, math.MaxUint32)
		return
	}
	e.u32(uint32(size))
	for i := 0; i < size; i++ {
		e.value(v[i])
	}
}

//TODO: handle nil value
func (e *encodeState) arr(v reflect.Value) {
	if v.Kind() != reflect.Array && v.Kind() != reflect.Slice {
		return
	}
	e.buf.WriteByte(AMF_STRICT_ARRAY)
	size := v.Len()
	if size > math.MaxUint32 {
		e.err = fmt.Errorf("amf0 encode array error: arr len %d is longer than %d.", size, math.MaxUint32)
		return
	}
	e.u32(uint32(size))
	for i := 0; i < size; i++ {
		e.envalue(v.Index(i))
	}
}

//TODO: handle nil value
func (e *encodeState) object(v reflect.Value) {
	if v.Kind() != reflect.Map || v.Type().Key().Kind() != reflect.String {
		return
	}
	e.buf.WriteByte(AMF_OBJECT)
	it := v.MapRange()
	for it.Next() {
		ik, iv := it.Key(), it.Value()
		size := ik.Len()
		if size > math.MaxUint16 {
			e.err = fmt.Errorf("amf0 encode object error: key len %d is longer than %d.", size, math.MaxUint16)
			return
		}
		e.u16(uint16(size))
		e.buf.WriteString(ik.String())
		e.envalue(iv)
	}
	e.buf.Write(objEnd)
}

func (e *encodeState) u32(v uint32) {
	e.cache = e.cache[:4]
	e.order.PutUint32(e.cache, v)
	e.buf.Write(e.cache)
}

func (e *encodeState) u16(v uint16) {
	e.cache = e.cache[:2]
	e.order.PutUint16(e.cache, v)
	e.buf.Write(e.cache)
}

func (e *encodeState) enstruct(v reflect.Value) {
	if v.Kind() != reflect.Struct {
		return
	}
	e.buf.WriteByte(AMF_OBJECT)
	fields := reflect.VisibleFields(v.Type())
	for _, f := range fields {
		var key string
		if key = f.Tag.Get("amf"); key == "" {
			key = f.Name
		}
		if key == "-" {
			continue
		}
		size := len(key)
		if size > math.MaxUint16 {
			e.err = fmt.Errorf("amf0 encode object error: key len %d is longer than %d.", size, math.MaxUint16)
			return
		}
		e.u16(uint16(size))
		e.buf.WriteString(key)
		e.envalue(v.FieldByIndex(f.Index))
	}
	e.buf.Write(objEnd)
}
