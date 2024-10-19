package lru

import "container/list"

type Cache struct {
	maxBytes int64      // 最大容量
	nBytes   int64      // 已使用的容量
	ll       *list.List // 待淘汰元素位于队首
	cache    map[string]*list.Element
	onRemove func(key string, value Value) // 某个元素被删除时的回调函数
}

type Value interface {
	Len() int
}

type entry struct {
	key   string
	value Value
}

func (e entry) size() int64 {
	return int64(len(e.key)) + int64(e.value.Len())
}

func New(maxBytes int64, onRemove func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes: maxBytes,
		nBytes:   0,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
		onRemove: onRemove,
	}
}

func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// 移除最近最少使用的元素
func (c *Cache) remove() {
	ele := c.ll.Front()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)

		// 更新cache信息
		delete(c.cache, kv.key)
		c.nBytes -= kv.size()
		if c.onRemove != nil {
			c.onRemove(kv.key, kv.value)
		}
	}
}

func (c *Cache) Put(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		e := &entry{key, value}
		ele := c.ll.PushBack(e)
		c.cache[key] = ele
		c.nBytes += e.size()
	}

	for c.maxBytes != 0 && c.nBytes > c.maxBytes {
		c.remove()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
