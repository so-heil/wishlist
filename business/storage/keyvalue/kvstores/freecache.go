package kvstores

import (
	"time"

	"github.com/coocood/freecache"
)

type FreeCache struct {
	fc *freecache.Cache
}

func NewFreeCache(size int) *FreeCache {
	fc := freecache.NewCache(size)
	return &FreeCache{fc}
}

func (fc *FreeCache) Set(key string, data []byte, expire time.Duration) error {
	return fc.fc.Set([]byte(key), data, int(expire.Seconds()))
}

func (fc *FreeCache) Get(key string) ([]byte, error) {
	return fc.fc.Get([]byte(key))
}
