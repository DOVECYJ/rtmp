package amf0

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// AMF0类型
const (
	AMF_NUMBER       = 0x00 //浮点数
	AMF_BOOLEAN      = 0x01 //布尔
	AMF_STRING       = 0x02 //字符串
	AMF_OBJECT       = 0x03 //键值对
	AMF_MOVIECLIP    = 0x04 //reserved
	AMF_NULL         = 0x05
	AMF_UNDEFINED    = 0x06
	AMF_REFERENCE    = 0x07
	AMF_ECMA_ARRAY   = 0x08 //键值对数组
	AMF_OBJECT_END   = 0x09 //对象结束
	AMF_STRICT_ARRAY = 0x0A //数组
	AMF_DATE         = 0x0B //日期
	AMF_LONG_STRING  = 0x0C //长字符串
	AMF_UNSUPPORTED  = 0x0D
	AMF_RECORDSET    = 0x0E //reserved
	AMF_XML_DOCUMENT = 0x0F
	AMF_TYPE_OBJECT  = 0x10 //类型对象
	AMF_AMF3         = 0x11
)

type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "amf: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Pointer {
		return "amf: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "amf: Unmarshal(nil " + e.Type.String() + ")"
}

type UnmarshalTypeError struct {
	Id   int
	Type reflect.Type
}

func (e *UnmarshalTypeError) Error() string {
	if e.Type == nil {
		return fmt.Sprintf("amf[%d]: Unmarshal(nil)", e.Id)
	}

	if e.Type.Kind() != reflect.Pointer {
		return fmt.Sprintf("amf[%d]: Unmarshal(non-pointer %s)", e.Id, e.Type)
	}
	return fmt.Sprintf("amf[%d]: Unmarshal(nil %s)", e.Id, e.Type)
}

type Amfkv map[string]any

type Amfarr []any

func (a Amfarr) Get(i int) any {
	return a[i]
}

func (a Amfarr) GetString(i int) (v string, ok bool) {
	v, ok = a[i].(string)
	return
}

func (a Amfarr) GetUint8(i int) (v uint8, ok bool) {
	var f float64
	if f, ok = a[i].(float64); ok {
		v = uint8(f)
	}
	return
}

func (a Amfarr) GetUint16(i int) (v uint16, ok bool) {
	var f float64
	if f, ok = a[i].(float64); ok {
		v = uint16(f)
	}
	return
}

func (a Amfarr) GetUint32(i int) (v uint32, ok bool) {
	var f float64
	if f, ok = a[i].(float64); ok {
		v = uint32(f)
	}
	return
}

func (a Amfarr) GetUint64(i int) (v uint64, ok bool) {
	var f float64
	if f, ok = a[i].(float64); ok {
		v = uint64(f)
	}
	return
}

func (a Amfarr) GetUint(i int) (v uint, ok bool) {
	var f float64
	if f, ok = a[i].(float64); ok {
		v = uint(f)
	}
	return
}

func (a Amfarr) GetInt8(i int) (v int8, ok bool) {
	var f float64
	if f, ok = a[i].(float64); ok {
		v = int8(f)
	}
	return
}

func (a Amfarr) GetInt16(i int) (v int16, ok bool) {
	var f float64
	if f, ok = a[i].(float64); ok {
		v = int16(f)
	}
	return
}

func (a Amfarr) GetInt32(i int) (v int32, ok bool) {
	var f float64
	if f, ok = a[i].(float64); ok {
		v = int32(f)
	}
	return
}

func (a Amfarr) GetInt64(i int) (v int64, ok bool) {
	var f float64
	if f, ok = a[i].(float64); ok {
		v = int64(f)
	}
	return
}

func (a Amfarr) GetInt(i int) (v int, ok bool) {
	var f float64
	if f, ok = a[i].(float64); ok {
		v = int(f)
	}
	return
}

func (a Amfarr) GetFloat32(i int) (v float32, ok bool) {
	var f float64
	if f, ok = a[i].(float64); ok {
		v = float32(f)
	}
	return
}

func (a Amfarr) GetFloat64(i int) (v float64, ok bool) {
	v, ok = a[i].(float64)
	return
}

func (a Amfarr) GetBool(i int) (v bool, ok bool) {
	v, ok = a[i].(bool)
	return
}

func (a Amfarr) GetKV(i int) (v Amfkv, ok bool) {
	v, ok = a[i].(Amfkv)
	return
}

func (a Amfarr) GetArr(i int) (v Amfarr, ok bool) {
	v, ok = a[i].(Amfarr)
	return
}

func (a Amfarr) PutString(i int, v *string) (ok bool) {
	*v, ok = a[i].(string)
	return
}

func (a Amfarr) PutUint8(i int, v *uint8) (ok bool) {
	*v, ok = a.GetUint8(i)
	return
}

func (a Amfarr) PutUint16(i int, v *uint16) (ok bool) {
	*v, ok = a.GetUint16(i)
	return
}

func (a Amfarr) PutUint32(i int, v *uint32) (ok bool) {
	*v, ok = a.GetUint32(i)
	return
}

func (a Amfarr) PutUint64(i int, v *uint64) (ok bool) {
	*v, ok = a.GetUint64(i)
	return
}

func (a Amfarr) PutUint(i int, v *uint) (ok bool) {
	*v, ok = a.GetUint(i)
	return
}

func (a Amfarr) PutInt8(i int, v *int8) (ok bool) {
	*v, ok = a.GetInt8(i)
	return
}

func (a Amfarr) PutInt16(i int, v *int16) (ok bool) {
	*v, ok = a.GetInt16(i)
	return
}

func (a Amfarr) PutInt32(i int, v *int32) (ok bool) {
	*v, ok = a.GetInt32(i)
	return
}

func (a Amfarr) PutInt64(i int, v *int64) (ok bool) {
	*v, ok = a.GetInt64(i)
	return
}

func (a Amfarr) PutInt(i int, v *int) (ok bool) {
	*v, ok = a.GetInt(i)
	return
}

func (a Amfarr) PutFloat32(i int, v *float32) (ok bool) {
	*v, ok = a.GetFloat32(i)
	return
}

func (a Amfarr) PutFloat64(i int, v *float64) (ok bool) {
	*v, ok = a[i].(float64)
	return
}

func (a Amfarr) PutBool(i int, v *bool) (ok bool) {
	*v, ok = a[i].(bool)
	return
}

func (a Amfarr) PutKV(i int, v *map[string]any) (ok bool) {
	*v, ok = a[i].(Amfkv)
	return
}

func (a Amfarr) PutArr(i int, v *[]any) (ok bool) {
	*v, ok = a[i].(Amfarr)
	return
}

// the amf kv method
func (a Amfkv) Get(key string) any {
	return a[key]
}

func (a Amfkv) GetBool(key string) (v bool, ok bool) {
	var r any
	if r, ok = a[key]; ok {
		v, ok = r.(bool)
	}
	return
}

func (a Amfkv) GetString(key string) (v string, ok bool) {
	var r any
	if r, ok = a[key]; ok {
		v, ok = r.(string)
	}
	return
}

func (a Amfkv) GetInt8(key string) (v int8, ok bool) {
	var f float64
	if f, ok = a.GetFloat64(key); ok {
		v = int8(f)
	}
	return
}

func (a Amfkv) GetInt16(key string) (v int16, ok bool) {
	var f float64
	if f, ok = a.GetFloat64(key); ok {
		v = int16(f)
	}
	return
}

func (a Amfkv) GetInt32(key string) (v int32, ok bool) {
	var f float64
	if f, ok = a.GetFloat64(key); ok {
		v = int32(f)
	}
	return
}

func (a Amfkv) GetInt64(key string) (v int64, ok bool) {
	var f float64
	if f, ok = a.GetFloat64(key); ok {
		v = int64(f)
	}
	return
}

func (a Amfkv) GetInt(key string) (v int, ok bool) {
	var f float64
	if f, ok = a.GetFloat64(key); ok {
		v = int(f)
	}
	return
}

func (a Amfkv) GetUint8(key string) (v uint8, ok bool) {
	var f float64
	if f, ok = a.GetFloat64(key); ok {
		v = uint8(f)
	}
	return
}

func (a Amfkv) GetUint16(key string) (v uint16, ok bool) {
	var f float64
	if f, ok = a.GetFloat64(key); ok {
		v = uint16(f)
	}
	return
}

func (a Amfkv) GetUint32(key string) (v uint32, ok bool) {
	var f float64
	if f, ok = a.GetFloat64(key); ok {
		v = uint32(f)
	}
	return
}

func (a Amfkv) GetUint64(key string) (v uint64, ok bool) {
	var f float64
	if f, ok = a.GetFloat64(key); ok {
		v = uint64(f)
	}
	return
}

func (a Amfkv) GetUint(key string) (v uint, ok bool) {
	var f float64
	if f, ok = a.GetFloat64(key); ok {
		v = uint(f)
	}
	return
}

func (a Amfkv) GetFloat32(key string) (v float32, ok bool) {
	var f float64
	if f, ok = a.GetFloat64(key); ok {
		v = float32(f)
	}
	return
}

func (a Amfkv) GetFloat64(key string) (v float64, ok bool) {
	var r any
	if r, ok = a[key]; ok {
		v, ok = r.(float64)
	}
	return
}

func (a Amfkv) GetKV(key string) (v Amfkv, ok bool) {
	var r any
	if r, ok = a[key]; ok {
		v, ok = r.(Amfkv)
	}
	return
}

func (a Amfkv) GetArr(key string) (v Amfarr, ok bool) {
	var r any
	if r, ok = a[key]; ok {
		v, ok = r.(Amfarr)
	}
	return
}

func (a *Amfarr) deValue(i int, v reflect.Value) error {
	if i >= len(*a) {
		return nil
	}
	switch v.Kind() {
	case reflect.Bool:
		if b, ok := a.GetBool(i); ok {
			v.SetBool(b)
		} else {
			return &UnmarshalTypeError{i, v.Type()}
		}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		if u, ok := a.GetUint64(i); ok {
			v.SetUint(u)
		} else {
			return &UnmarshalTypeError{i, v.Type()}
		}
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		if u, ok := a.GetInt64(i); ok {
			v.SetInt(u)
		} else {
			return &UnmarshalTypeError{i, v.Type()}
		}
	case reflect.Float32, reflect.Float64:
		if f, ok := a.GetFloat64(i); ok {
			v.SetFloat(f)
		} else {
			return &UnmarshalTypeError{i, v.Type()}
		}
	case reflect.String:
		if s, ok := a.GetString(i); ok {
			v.SetString(s)
		} else {
			return &UnmarshalTypeError{i, v.Type()}
		}
	case reflect.Array:
		if sa, ok := a.GetArr(i); ok {
			if err := sa.deArray(v); err != nil {
				return err
			}
		} else {
			return &UnmarshalTypeError{i, v.Type()}
		}
	case reflect.Slice:
		if sa, ok := a.GetArr(i); ok {
			if err := sa.deSlice(v); err != nil {
				return err
			}
		} else {
			return &UnmarshalTypeError{i, v.Type()}
		}
	case reflect.Map:
		if kv, ok := a.GetKV(i); ok {
			if err := kv.deMap(v); err != nil {
				return err
			}
		} else {
			return &UnmarshalTypeError{i, v.Type()}
		}
	case reflect.Struct:
		if kv, ok := a.GetKV(i); ok {
			if err := kv.deStruct(v); err != nil {
				return err
			}
		} else {
			return &UnmarshalTypeError{i, v.Type()}
		}
	case reflect.Pointer:
		fmt.Println("pointer")
	case reflect.Interface:
		v.Set(reflect.ValueOf(a.Get(i)))
	}
	return nil
}

func (a *Amfarr) deArray(v reflect.Value) (err error) {
	if v.Kind() != reflect.Array {
		return &InvalidUnmarshalError{v.Type()}
	}
	size := min(len(*a), v.Len())
	for i := 0; i < size; i++ {
		if err = a.deValue(i, v.Index(i)); err != nil {
			return
		}
	}
	return
}

func (a *Amfarr) deSlice(v reflect.Value) (err error) {
	if v.Kind() != reflect.Slice {
		return &InvalidUnmarshalError{v.Type()}
	}
	size := len(*a)
	if v.IsNil() {
		v.Set(reflect.MakeSlice(v.Type(), size, size))
	}
	if size > v.Len() {
		size = v.Len()
	}
	for i := 0; i < size; i++ {
		if err = a.deValue(i, v.Index(i)); err != nil {
			return
		}
	}
	return
}

func (a *Amfarr) deStruct(v reflect.Value) (err error) {
	if v.Kind() != reflect.Struct {
		return &InvalidUnmarshalError{v.Type()}
	}
	sf := reflect.VisibleFields(v.Type())
	for _, f := range sf {
		var id int
		tag := f.Tag.Get("amf")
		if tag == "-" {
			continue
		}
		if tag != "" {
			if n, e := strconv.Atoi(tag); e == nil {
				id = n
			} else {
				id = f.Index[0]
			}

		} else {
			id = f.Index[0]
		}
		if id >= len(*a) {
			continue
		}
		a.deValue(id, v.FieldByIndex(f.Index))
	}
	return
}

func (a *Amfkv) deStruct(v reflect.Value) (err error) {
	if v.Kind() != reflect.Struct {
		return &InvalidUnmarshalError{v.Type()}
	}
	sf := reflect.VisibleFields(v.Type())
	for _, f := range sf {
		key := f.Tag.Get("amf")
		if key == "-" {
			continue
		}
		if key != "" {
			if _, ok := (*a)[key]; !ok {
				continue
			}
		} else {
			key = f.Name
		}
		a.deValue(key, v.FieldByIndex(f.Index))
	}
	return
}

func (a *Amfkv) deMap(v reflect.Value) (err error) {
	if v.Kind() != reflect.Map || v.Type().Key().Kind() != reflect.String {
		return &InvalidUnmarshalError{v.Type()}
	}
	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}
	for sk, sv := range *a {
		v.SetMapIndex(reflect.ValueOf(sk), reflect.ValueOf(sv))
	}
	return
}

func (a *Amfkv) deValue(key string, v reflect.Value) (err error) {
	var exist bool
	if key, exist = a.exist(key); !exist {
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		if b, ok := a.GetBool(key); ok {
			v.SetBool(b)
		} else {
			return &InvalidUnmarshalError{v.Type()}
		}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		if u, ok := a.GetUint64(key); ok {
			v.SetUint(u)
		} else {
			return &InvalidUnmarshalError{v.Type()}
		}
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		if u, ok := a.GetInt64(key); ok {
			v.SetInt(u)
		} else {
			return &InvalidUnmarshalError{v.Type()}
		}
	case reflect.Float32, reflect.Float64:
		if f, ok := a.GetFloat64(key); ok {
			v.SetFloat(f)
		} else {
			return &InvalidUnmarshalError{v.Type()}
		}
	case reflect.String:
		if s, ok := a.GetString(key); ok {
			v.SetString(s)
		} else {
			return &InvalidUnmarshalError{v.Type()}
		}
	case reflect.Array:
		if sa, ok := a.GetArr(key); ok {
			if err := sa.deArray(v); err != nil {
				return err
			}
		} else {
			return &InvalidUnmarshalError{v.Type()}
		}
	case reflect.Slice:
		if sa, ok := a.GetArr(key); ok {
			if err := sa.deSlice(v); err != nil {
				return err
			}
		} else {
			return &InvalidUnmarshalError{v.Type()}
		}
	case reflect.Map:
		if kv, ok := a.GetKV(key); ok {
			if err := kv.deMap(v); err != nil {
				return err
			}
		} else {
			return &InvalidUnmarshalError{v.Type()}
		}
	case reflect.Struct:
		if kv, ok := a.GetKV(key); ok {
			if err := kv.deStruct(v); err != nil {
				return err
			}
		} else {
			return &InvalidUnmarshalError{v.Type()}
		}
	case reflect.Pointer:
		fmt.Println("pointer")
	case reflect.Interface:
		v.Set(reflect.ValueOf(a.Get(key)))
	}
	return nil
}

func (a *Amfkv) exist(key string) (nkey string, exist bool) {
	for k, _ := range *a {
		if k == key {
			return key, true
		} else if strings.EqualFold(k, key) {
			return k, true
		}
	}
	return key, false
}

func Unmarshal(data []byte, v ...any) error {
	var d decodeState
	return d.init(data).unmarshal(v...)
}

func Decode(data []byte) (ar Amfarr, err error) {
	var d decodeState
	return d.init(data).decode()
}

type decodeState struct {
	amftype uint8
	count   int
	offset  int
	data    []byte
	order   binary.ByteOrder
	err     error
}

func (d *decodeState) init(data []byte) *decodeState {
	d.data = data
	d.amftype = 0xFF
	d.order = binary.BigEndian
	return d
}

// amf反序列化
func (d *decodeState) unmarshal(v ...any) error {
	var size int
	if size = len(v); size == 0 {
		return nil
	}
	ar, err := d.decode()
	if err != nil {
		return err
	}
	if size > len(ar) {
		size = len(ar)
	}
	if size == 0 {
		return nil
	}
	// 将amf数组解析到单个结构体
	if len(v) == 1 {
		if v[0] == nil {
			return nil
		}
		rv := reflect.ValueOf(v[0])
		if rv.Kind() != reflect.Pointer || rv.IsNil() {
			return &InvalidUnmarshalError{rv.Type()}
		}
		e := rv.Elem()
		if e.Kind() == reflect.Struct {
			pv := indirect(rv, true)
			err := ar.deStruct(pv)
			return err
		} else if e.Kind() == reflect.Slice {
			pv := indirect(rv, true)
			err := ar.deSlice(pv)
			return err
		}
	}
	// 将amf数组解析到多个结果
	for i := 0; i < size; i++ {
		if v[i] == nil {
			continue
		}
		rv := reflect.ValueOf(v[i])
		if rv.Kind() != reflect.Pointer || rv.IsNil() {
			return &InvalidUnmarshalError{rv.Type()}
		}
		pv := indirect(rv, true)
		if err := ar.deValue(i, pv); err != nil {
			return err
		}
	}
	return nil
}

func (d *decodeState) decode() (Amfarr, error) {
	var ar Amfarr
	for d.offset < len(d.data) {
		v := d.value()
		if d.err != nil {
			return nil, d.err
		}
		ar = append(ar, v)
	}
	return ar, d.err
}

func (d *decodeState) scanAmfType() {
	if d.err != nil || d.offset >= len(d.data) {
		return
	}
	d.amftype = uint8(d.data[d.offset])
	d.offset++
}

func (d *decodeState) value() any {
	d.scanAmfType()
	switch d.amftype {
	default:
		d.err = fmt.Errorf("Unsupport amf type: %d", d.amftype)
		return nil
	case AMF_UNDEFINED, AMF_REFERENCE, AMF_DATE,
		AMF_UNSUPPORTED, AMF_XML_DOCUMENT, AMF_TYPE_OBJECT,
		AMF_AMF3, AMF_MOVIECLIP, AMF_RECORDSET:
		d.err = fmt.Errorf("Unimplate amf type: %d", d.amftype)
		return nil
	case AMF_NUMBER:
		return d.number()
	case AMF_BOOLEAN:
		return d.tf()
	case AMF_STRING:
		return d.str()
	case AMF_OBJECT:
		return d.object()
	case AMF_ECMA_ARRAY:
		return d.ecmarr()
	case AMF_STRICT_ARRAY:
		return d.arr()
	case AMF_LONG_STRING:
		return d.longstr()
	case AMF_NULL:
		return nil
	}
}

func (d *decodeState) number() float64 {
	start := d.offset
	d.offset += 8
	bs := d.order.Uint64(d.data[start:d.offset])
	return math.Float64frombits(bs)
}

func (d *decodeState) tf() bool {
	start := d.offset
	d.offset += 1
	return d.data[start] != 0
}

func (d *decodeState) str() string {
	start := d.offset
	d.offset += 2
	size := d.order.Uint16(d.data[start:d.offset])
	start = d.offset
	d.offset += int(size)
	return string(d.data[start:d.offset])
}

func (d *decodeState) longstr() string {
	start := d.offset
	d.offset += 4
	size := d.order.Uint32(d.data[start:d.offset])
	start = d.offset
	d.offset += int(size)
	return string(d.data[start:d.offset])
}

func (d *decodeState) object() Amfkv {
	m := make(map[string]any)
	for d.u24() != AMF_OBJECT_END {
		d.unread(3)
		key := d.str()
		val := d.value()
		m[key] = val
	}
	return Amfkv(m)
}

func (d *decodeState) ecmarr() Amfkv {
	size := d.u32()
	m := make(map[string]any, size)
	for i := uint32(0); i < size; i++ {
		key := d.str()
		val := d.value()
		m[key] = val
	}
	if d.u24() != AMF_OBJECT_END {
		d.err = fmt.Errorf("Expect AMF Object End(00 00 09), but(%x)", d.data[d.offset-3:d.offset])
		return nil
	}
	return Amfkv(m)
}

func (d *decodeState) arr() Amfarr {
	size := d.u32()
	arr := make([]any, size)
	for i := uint32(0); i < size; i++ {
		arr[i] = d.value()
	}
	return Amfarr(arr)
}

func (d *decodeState) u16() uint16 {
	start := d.offset
	d.offset += 2
	return d.order.Uint16(d.data[start:d.offset])
}

func (d *decodeState) u24() uint32 {
	start := d.offset
	d.offset += 3
	return uint32(d.data[start]) | uint32(d.data[start+1]) | uint32(d.data[start+2])
}

func (d *decodeState) u32() uint32 {
	start := d.offset
	d.offset += 4
	return d.order.Uint32(d.data[start:d.offset])
}

func (d *decodeState) unread(n int) {
	d.offset -= n
	if d.offset < 0 {
		d.offset = 0
	}
}

// 检查数据完整性
func (d *decodeState) checkIntegrity(n int) {
	if d.err != nil {
		return
	}
	if d.offset+n >= len(d.data) {
		d.err = fmt.Errorf("")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// indirect walks down v allocating pointers as needed,
// until it gets to a non-pointer.
// If decodingNull is true, indirect stops at the first settable pointer so it
// can be set to nil.
func indirect(v reflect.Value, decodingNull bool) reflect.Value {
	v0 := v
	haveAddr := false

	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Pointer && v.Type().Name() != "" && v.CanAddr() {
		haveAddr = true
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Pointer && !e.IsNil() && (!decodingNull || e.Elem().Kind() == reflect.Pointer) {
				haveAddr = false
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Pointer {
			break
		}

		if decodingNull && v.CanSet() {
			break
		}

		// Prevent infinite loop if v is an interface pointing to its own address:
		//     var v interface{}
		//     v = &v
		if v.Elem().Kind() == reflect.Interface && v.Elem().Elem() == v {
			v = v.Elem()
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		if haveAddr {
			v = v0 // restore original value after round-trip Value.Addr().Elem()
			haveAddr = false
		} else {
			v = v.Elem()
		}
	}
	return v
}

type Decoder interface {
	GetBool() (bool, bool)
	GetInt8() (int8, bool)
	GetInt16() (int16, bool)
	GetInt32() (int32, bool)
	GetInt64() (int64, bool)
	GetInt() (int, bool)
	GetUint8() (uint8, bool)
	GetUint16() (uint16, bool)
	GetUint32() (uint32, bool)
	GetUint64() (uint64, bool)
	GetUint() (uint, bool)
	GetFloat32() (float32, bool)
	GetFloat64() (float64, bool)
	GetString() (string, bool)

	Decode(v ...any) error
	Skip() Decoder
}

func NewDecoder(data []byte) (Decoder, error) {
	if ar, err := Decode(data); err != nil {
		return nil, err
	} else {
		return &decodeResult{ar: ar}, nil
	}
}

type decodeResult struct {
	ar    Amfarr
	index int
	last  int
}

func (d *decodeResult) GetBool() (bool, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetBool(d.last)
}

func (d *decodeResult) GetInt8() (int8, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetInt8(d.last)
}

func (d *decodeResult) GetInt16() (int16, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetInt16(d.last)
}

func (d *decodeResult) GetInt32() (int32, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetInt32(d.last)
}

func (d *decodeResult) GetInt64() (int64, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetInt64(d.last)
}

func (d *decodeResult) GetInt() (int, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetInt(d.last)
}

func (d *decodeResult) GetUint8() (uint8, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetUint8(d.last)
}

func (d *decodeResult) GetUint16() (uint16, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetUint16(d.last)
}

func (d *decodeResult) GetUint32() (uint32, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetUint32(d.last)
}

func (d *decodeResult) GetUint64() (uint64, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetUint64(d.last)
}

func (d *decodeResult) GetUint() (uint, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetUint(d.last)
}

func (d *decodeResult) GetFloat32() (float32, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetFloat32(d.last)
}

func (d *decodeResult) GetFloat64() (float64, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetFloat64(d.last)
}

func (d *decodeResult) GetString() (string, bool) {
	d.last = d.index
	d.index++
	return d.ar.GetString(d.last)
}

func (d *decodeResult) Decode(v ...any) error {
	remain := len(d.ar) - d.index
	size := min(remain, len(v))
	if size == 0 {
		return nil
	}

	for i := 0; i < size; i++ {
		if v[i] != nil {
			rv := reflect.ValueOf(v[i])
			if rv.Kind() != reflect.Pointer || rv.IsNil() {
				return &InvalidUnmarshalError{rv.Type()}
			}
			pv := indirect(rv, true)
			err := d.ar.deValue(d.index, pv)
			if err != nil {
				return err
			}
		}
		d.last = d.index
		d.index++
	}
	return nil
}

func (d *decodeResult) Skip() Decoder {
	d.last = d.index
	d.index++
	return d
}
