package lru

import "container/list"

// Cache 缓存淘汰策略：最近最久未使用LRU
type Cache struct {
	maxBytes  int64 //最大缓存字节数
	nBytes    int64
	ll        *list.List
	cache     map[string]*list.Element
	onEvicted func(key string, value Value) //删除时的回调函数
}

// Value value都实现了这个接口 方便计算存入值所占字节数
type Value interface {
	Len() int
}

type Entry struct {
	key   string //存key是为了方便从cache中移除
	value Value
}

func New(maxBytes int64, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string) (Value, bool) {
	if elem, ok := c.cache[key]; ok {
		kv := elem.Value.(*Entry)
		c.ll.MoveToBack(elem)
		return kv.value, true
	}
	return nil, false
}

// RemoveOldest 缓存淘汰
func (c *Cache) RemoveOldest() {
	elem := c.ll.Front()
	if elem != nil {
		c.ll.Remove(elem)
		kv := elem.Value.(*Entry)
		delete(c.cache, kv.key)
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.onEvicted != nil {
			c.onEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) Add(key string, value Value) {
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToBack(elem)
		kv := elem.Value.(*Entry)
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		elem = c.ll.PushBack(&Entry{key: key, value: value})
		c.cache[key] = elem
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.nBytes > c.maxBytes {
		c.RemoveOldest()
	}
}
