package keyvalue

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("key is not present in store")
)

type KeyValueStore interface {
	Set(key string, data []byte, expire time.Duration) error
	Get(key string) ([]byte, error)
	Del(key string) bool
}
