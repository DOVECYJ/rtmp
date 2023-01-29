package amf0

import (
	"reflect"
	"testing"
)

var (
	data = []byte{
		0x02, 0x00, 0x07, 0x63, 0x6F, 0x6E, 0x6E, 0x65, 0x63, 0x74, 0x00, 0x3F, 0xF0, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x03, 0x00, 0x03, 0x61, 0x70, 0x70, 0x02, 0x00, 0x04, 0x6C, 0x69, 0x76, 0x65,
		0x00, 0x04, 0x74, 0x79, 0x70, 0x65, 0x02, 0x00, 0x0A, 0x6E, 0x6F, 0x6E, 0x70, 0x72, 0x69, 0x76,
		0x61, 0x74, 0x65, 0x00, 0x08, 0x66, 0x6C, 0x61, 0x73, 0x68, 0x56, 0x65, 0x72, 0x02, 0x00, 0x24,
		0x46, 0x4D, 0x4C, 0x45, 0x2F, 0x33, 0x2E, 0x30, 0x20, 0x28, 0x63, 0x6F, 0x6D, 0x70, 0x61, 0x74,
		0x69, 0x62, 0x6C, 0x65, 0x3B, 0x20, 0x4C, 0x61, 0x76, 0x66, 0x35, 0x39, 0x2E, 0x33, 0x34, 0x2E,
		0x31, 0x30, 0x32, 0x29, 0x00, 0x05, 0x74, 0x63, 0x55, 0x72, 0x6C, 0x02, 0x00, 0x1A, 0x72, 0x74,
		0x6D, 0x70, 0x3A, 0x2F, 0x2F, 0x6C, 0x6F, 0x63, 0x61, 0x6C, 0x68, 0x6F, 0x73, 0x74, 0x3A, 0x31,
		0x39, 0x33, 0x35, 0x2F, 0x6C, 0x69, 0x76, 0x65, 0x00, 0x00, 0x09,
	}
)

type CommandMsg struct {
	Name string
	Tid  int
	Obj  map[string]any
}

func TestDecode(t *testing.T) {
	ar, err := Decode(data)
	if err != nil {
		t.Fail()
	}
	t.Logf("amfarr: %v", ar)
}

func TestUnmarshal(t *testing.T) {
	var v CommandMsg
	err := Unmarshal(data, &v.Name, &v.Tid, &v.Obj)
	if err != nil {
		t.Errorf("unmarshal error: %v", err)
	}
	t.Logf("CommandMsg: %+v", v)
}

func TestUnmarshalArrStruct(t *testing.T) {
	var v CommandMsg
	err := Unmarshal(data, &v)
	if err != nil {
		t.Errorf("unmarshal struct error: %v", err)
	}
	t.Logf("CommandMsg: %+v", v)
}

func TestUnmarshalBool(t *testing.T) {
	ar := Amfarr([]any{true})
	d := decodeResult{ar: ar}
	var b bool
	err := d.Decode(&b)
	if err != nil {
		t.Errorf("decode bool error:%v", err)
	}
	if b != true {
		t.FailNow()
	}
	t.Log("decode bool:", b)
}

func TestUnmarshalNumber(t *testing.T) {
	ar := Amfarr([]any{1.0, 2.0, 3.3})
	d := decodeResult{ar: ar}
	var i int
	var u uint
	var f float32
	err := d.Decode(&i, &u, &f)
	if err != nil {
		t.Errorf("decode num error:%v", err)
	}
	if i != 1 || u != 2 || f != 3.3 {
		t.FailNow()
	}
	t.Log("decode num:", i, u, f)
}

func TestUnmarshalString(t *testing.T) {
	ar := Amfarr([]any{"hello"})
	d := decodeResult{ar: ar}
	var s string
	err := d.Decode(&s)
	if err != nil {
		t.Errorf("decode string error:%v", err)
	}
	if s != "hello" {
		t.FailNow()
	}
	t.Log("decode string:", s)
}

func TestUnmarshalArray(t *testing.T) {
	ar := Amfarr([]any{Amfarr([]any{1.0, 2.0, 3.0})})
	d := decodeResult{ar: ar}
	var a [3]int
	err := d.Decode(&a)
	if err != nil {
		t.Errorf("decode array error:%v", err)
	}
	t.Log("decode array:", a)
	if a != [3]int{1, 2, 3} {
		t.FailNow()
	}
}

func TestUnmarshalSlice(t *testing.T) {
	ar := Amfarr([]any{Amfarr([]any{1.0, 2.0, 3.0})})
	d := decodeResult{ar: ar}
	var a []int
	err := d.Decode(&a)
	if err != nil {
		t.Errorf("decode slice error:%v", err)
	}
	t.Log("decode slice:", a)
	if !reflect.DeepEqual(a, []int{1, 2, 3}) {
		t.FailNow()
	}
}

func TestUnmarshalInterface(t *testing.T) {
	ar := Amfarr([]any{"hello"})
	d := decodeResult{ar: ar}
	var a any
	err := d.Decode(&a)
	if err != nil {
		t.Errorf("decode interface error:%v", err)
	}
	t.Log("decode interface:", a)
	if !reflect.DeepEqual(a, "hello") {
		t.FailNow()
	}
}

func TestUnmarshalStruct(t *testing.T) {
	type User struct {
		Id   int `amf:"num"`
		Name string
	}
	ar := Amfarr([]any{Amfkv(map[string]any{"num": 1.0, "name": "name"})})
	d := decodeResult{ar: ar}
	var a User
	err := d.Decode(&a)
	if err != nil {
		t.Errorf("decode struct error:%v", err)
	}
	t.Log("decode struct:", a)
	if !reflect.DeepEqual(a, User{1, "name"}) {
		t.FailNow()
	}
}

func TestUnmarshalMap(t *testing.T) {
	ar := Amfarr([]any{Amfkv(map[string]any{"a": 1})})
	d := decodeResult{ar: ar}
	var a map[string]int
	err := d.Decode(&a)
	if err != nil {
		t.Errorf("decode map error:%v", err)
	}
	t.Log("decode map:", a)
	if !reflect.DeepEqual(a, map[string]int{"a": 1}) {
		t.FailNow()
	}
}
