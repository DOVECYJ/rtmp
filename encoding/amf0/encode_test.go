package amf0

import "testing"

func TestMarshalString(t *testing.T) {
	v := "a"
	bs, err := Marshal(v)
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	t.Logf("marshal string: %x", bs)
}

func TestMarshalNumber(t *testing.T) {
	v := 1
	bs, err := Marshal(v)
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	t.Logf("marshal number: %x", bs)
}

func TestMarshalBool(t *testing.T) {
	v := true
	bs, err := Marshal(v)
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	t.Logf("marshal bool: %x", bs)
}

func TestMarshalKV(t *testing.T) {
	v := map[string]any{
		"a": 1,
		"b": "yes",
	}
	bs, err := Marshal(v)
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	t.Logf("marshal kv: %x", bs)
}

func TestMarshalArr(t *testing.T) {
	v := []any{1, "a"}
	bs, err := Marshal(v)
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	t.Logf("marshal arr: %x", bs)
}

func TestMarshalMap(t *testing.T) {
	v := map[string]int{"a": 1, "b": 2}
	bs, err := Marshal(v)
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	t.Logf("marshal map: %x", bs)
}

func TestMarshalStruct(t *testing.T) {
	type Kv struct {
		key   string
		value int `amf:"val"`
	}
	kv := Kv{"a", 1}
	bs, err := Marshal(kv)
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	t.Logf("marshal struct: %x", bs)
}

func TestEncode(t *testing.T) {
	bs, err := Encode("a", 1)
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	t.Logf("encode: %x", bs)
}
