package hash

import (
	"hash/crc32"
	"slices"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash
	replicas int      // 虚拟节点倍数
	nodes    []uint32 // 储存所有节点(真实/虚拟)哈希值，有序
	hashMap  map[uint32]string
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[uint32]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}

	return m
}

func (m *Map) Add(nodes ...string) {
	for _, node := range nodes {
		for i := 0; i < m.replicas; i++ {
			hsh := m.hash([]byte(strconv.Itoa(i) + node))
			m.nodes = append(m.nodes, hsh)
			m.hashMap[hsh] = node
		}
	}
	slices.Sort(m.nodes)
}

func (m *Map) Get(key string) string {
	if len(m.nodes) == 0 {
		return ""
	}

	hsh := m.hash([]byte(key))
	idx := sort.Search(len(m.nodes), func(i int) bool {
		return m.nodes[i] >= hsh
	})

	return m.hashMap[m.nodes[idx%len(m.nodes)]]
}
