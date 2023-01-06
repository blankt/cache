package cache

import (
	"fmt"
	"log"
	"testing"
)

var db = map[string]string{
	"tlf": "630",
	"pxy": "589",
	"xm":  "567",
}

func TestGroup_Get(t *testing.T) {
	loadCount := make(map[string]int)
	cache := NewGroup("test", 2<<10, GetterFunc(func(key string) ([]byte, error) {
		log.Println("from db load ", key)
		if v, ok := db[key]; ok {
			if _, ok := loadCount[key]; !ok {
				loadCount[key] = 0
			}
			loadCount[key] += 1
			return []byte(v), nil
		}
		return nil, fmt.Errorf("not found this data %s", key)
	}))

	for k, v := range db {
		if data, err := cache.Get(k); err != nil || data.String() != v {
			t.Fatalf("get cache error :%v", v)
		}
		if _, err := cache.Get(k); err != nil || loadCount[k] > 1 {
			t.Fatalf("cache miss :%v", v)
		}
	}

	if data, err := cache.Get("not exist"); err == nil || data.String() != "" {
		t.Fatalf("get cache error")
	}
}
