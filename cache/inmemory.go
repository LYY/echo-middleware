package cache

import (
	"reflect"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

type InMemoryStore struct {
	gocache.Cache
}

func NewInMemoryStore(defaultExpiration time.Duration) *InMemoryStore {
	return &InMemoryStore{*gocache.New(defaultExpiration, time.Minute)}
}

func (c *InMemoryStore) Get(key string, value interface{}) error {
	val, found := c.Cache.Get(key)
	if !found {
		return ErrCacheMiss
	}

	v := reflect.ValueOf(value)
	if v.Type().Kind() == reflect.Ptr && v.Elem().CanSet() {
		v.Elem().Set(reflect.ValueOf(val))
		return nil
	}
	return ErrNotStored
}

func (c *InMemoryStore) Set(key string, value interface{}, expires time.Duration) error {
	// NOTE: go-cache understands the values of DEFAULT and FOREVER
	c.Cache.Set(key, value, expires)
	return nil
}

func (c *InMemoryStore) Add(key string, value interface{}, expires time.Duration) error {
	err := c.Cache.Add(key, value, expires)
	return covertInMemCacheError(err)
}

func (c *InMemoryStore) Replace(key string, value interface{}, expires time.Duration) error {
	if err := c.Cache.Replace(key, value, expires); err != nil {
		return ErrNotStored
	}
	return nil
}

func (c *InMemoryStore) Delete(key string) error {
	c.Cache.Delete(key)
	return nil
}

func (c *InMemoryStore) Increment(key string, n int64) (int64, error) {
	err := c.Cache.Increment(key, n)
	if err != nil {
		return 0, covertInMemCacheError(err)
	}
	newValue, _ := c.Cache.Get(key)
	return newValue.(int64), nil
}

func (c *InMemoryStore) Decrement(key string, n int64) (int64, error) {
	err := c.Cache.Decrement(key, n)
	if err != nil {
		return 0, covertInMemCacheError(err)
	}
	newValue, _ := c.Cache.Get(key)
	return newValue.(int64), nil
}

func (c *InMemoryStore) Flush() error {
	c.Cache.Flush()
	return nil
}

func covertInMemCacheError(err error) error {
	if err != nil {
		msg := err.Error()
		if strings.HasSuffix(msg, "doesn't exist") {
			return ErrCacheMiss
		} else if strings.HasSuffix(msg, "already exists") {
			return ErrNotStored
		} else if strings.HasSuffix(msg, "not found") {
			return ErrCacheMiss
		}
	}
	return err
}
