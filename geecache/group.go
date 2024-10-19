package geecache

import (
	"fmt"
	"geecache/cache"
	"geecache/internal/singleflight"
	"geecache/pb"
	"log"
	"sync"
)

var (
	mu     sync.RWMutex // 对groups的读多写少
	groups = make(map[string]*Group)
)

// A Getter loads data for a key.
type Getter interface {
	Get(key string) ([]byte, error)
}

// A GetterFunc implements Getter with a function.
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// 可以根据业务对cache分组

type Group struct {
	name      string
	mainCache *cache.Cache
	peers     PeerPicker
	getter    Getter // 缓存未命中时，调用getter从本地读取(如数据库)
	loader    *singleflight.Group
}

func NewGroup(name string, maxBytes int64, getter Getter) *Group {
	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		mainCache: cache.New(maxBytes, nil),
		getter:    getter,
		loader:    new(singleflight.Group),
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("peers already exist!")
	}
	g.peers = peers
}

func (g *Group) Get(key string) ([]byte, error) {
	if key == "" {
		return []byte{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.Get(key); ok {
		log.Printf("[Cache] key %s hit", key)
		return v, nil
	}

	log.Printf("[Cache] key %s miss,load it", key)
	return g.load(key)
}

func (g *Group) load(key string) ([]byte, error) {
	// 瞬间大量并发请求时,只执行一次
	bi, err := g.loader.Do(key, func() (any, error) {
		// 尝试从远程获取
		if g.peers != nil {
			if pg := g.peers.PickPeer(key); pg != nil {
				log.Printf("pg=%+v", pg)
				// 远程获取成功
				req := pb.Request{
					Group: g.name,
					Key:   key,
				}
				var resp pb.Response
				err := pg.Get(&req, &resp)
				if err != nil {
					log.Printf("[Cache] failed to get %s from peer:%s", key, err.Error())
				}
				return resp.Value, err
			}
		}

		log.Printf("Try to load %s locally", key)
		// 从本地获取
		return g.getLocally(key)
	})

	return bi.([]byte), err
}

func (g *Group) getLocally(key string) (b []byte, err error) {
	b, err = g.getter.Get(key)
	if err == nil {
		g.populate(key, b)
	}
	return
}

func (g *Group) populate(key string, value []byte) {
	g.mainCache.Put(key, value)
}
