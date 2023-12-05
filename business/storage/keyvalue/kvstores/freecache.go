package kvstores

import (
	"errors"
	"time"

	"github.com/coocood/freecache"
	"github.com/so-heil/wishlist/business/storage/keyvalue"
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
	res, err := fc.fc.Get([]byte(key))
	if err != nil {
		if errors.Is(err, freecache.ErrNotFound) {
			return nil, keyvalue.ErrNotFound
		}
		return nil, err
	}
	return res, nil
}

func (fc *FreeCache) Del(key string) bool {
	return fc.fc.Del([]byte(key))
}
