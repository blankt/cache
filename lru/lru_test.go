package lru

import (
	"reflect"
	"testing"
)

type String string

func (d String) Len() int {
	return len(d)
}

func TestCache_Get(t *testing.T) {
	lru := New(int64(0), nil)
	lru.Add("key1", String("112233"))
	if value, ok := lru.Get("key1"); !ok || string(value.(String)) != "112233" {
		t.Fatalf("命中缓存失败")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("key2 根本不存在")
	}
}

func TestCache_RemoveOldest(t *testing.T) {
	key1, key2, key3 := "key1", "key2", "key3"
	value1, value2, value3 := "value1", "value2", "value"
	size := len(key1) + len(key2) + len(value1) + len(value2)
	lru := New(int64(size), nil)
	lru.Add(key1, String(value1))
	lru.Add(key2, String(value2))
	lru.Add(key3, String(value3))
	if v, ok := lru.Get(key1); ok {
		t.Fatalf("key1：%v 不应该存在缓存中", v)
	}
	if lru.nBytes > lru.maxBytes {
		t.Fatalf("缓存大小出错")
	}
}

func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	lru := New(int64(10), callback)
	lru.Add("key1", String("123456"))
	lru.Add("k2", String("k2"))
	lru.Add("k3", String("k3"))
	lru.Add("k4", String("k4"))

	expect := []string{"key1", "k2"}

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("回调函数执行出错 %s", expect)
	}
}
