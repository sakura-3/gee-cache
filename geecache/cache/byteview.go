package cache

import (
	"geecache/internal/lru"
)

// Read-Only

type byteView struct {
	b []byte
}

var _ lru.Value = (*byteView)(nil)

func (bv byteView) Len() int {
	return len(bv.b)
}

func (bv byteView) String() string {
	return string(bv.b)
}

func (bv byteView) Bytes() []byte {
	r := make([]byte, len(bv.b))
	copy(r, bv.b)
	return r
}
