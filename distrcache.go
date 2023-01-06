package cache

import (
	"fmt"
	"log"
	"sync"

	"cache/cachepb"
	"cache/singleflight"
)

type Group struct {
	name      string
	mainCache cache
	getter    Getter
	peers     PeerPicker
	loader    *singleflight.Group
}

var (
	mu     sync.RWMutex
	Groups = make(map[string]*Group)
)

// Getter 定义回调函数接口 当数据不存在时 用户可自定义数据源获取数据
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 接口型函数
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil getter")
	}
	mu.Lock()
	defer mu.Unlock()
	group := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	Groups[name] = group
	return group
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	g := Groups[name]
	return g
}

// RegisterPeers 注入httpPool的方法
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		log.Println("exist peers")
	}
	g.peers = peers
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is null")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("cache hit")
		return v, nil
	}

	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	data, err := g.loader.Call(key, func() (interface{}, error) {
		if g.peers != nil {
			if peerGetter, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peerGetter, key); err == nil {
					return value, nil
				}
				log.Printf("get cache fail %v", key)
			}
		}

		return g.getLocally(key)
	})
	if err != nil {
		return data.(ByteView), err
	}
	return
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	in := &cachepb.Request{
		Group: g.name,
		Key:   key,
	}
	out := &cachepb.Response{}
	err := peer.Get(in, out)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: out.Value}, nil
}

func (g *Group) getLocally(key string) (value ByteView, err error) {
	b, err := g.getter.Get(key)
	if err != nil {
		return
	}

	//防止被外部修改 用cloneBytes
	value = ByteView{b: cloneBytes(b)}
	g.populateCache(key, ByteView{b: b})
	return
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

func (g *Group) Add(key string, value ByteView) {
	g.mainCache.add(key, value)
}
