package cache

import (
	"errors"
	"sync"
	"time"
)

type Cache struct {
	data       map[string]*cacheItem
	defaultExp time.Duration
	check      time.Duration
	mu         *sync.RWMutex
}

type cacheItem struct {
	val        interface{}
	expiration time.Time
}

func NewCache(exp, check time.Duration) *Cache {
	return &Cache{
		data:       make(map[string]*cacheItem),
		defaultExp: exp,
		check:      check,
		mu:         &sync.RWMutex{},
	}
}

func (c *Cache) StartClearing() {
	go c.clearing()
}

func (c *Cache) clearing() {
	ticker := time.NewTicker(c.check)
	select {
	case <-ticker.C:
		dataToClear := make([]string, 0)
		for key, item := range c.data {
			if item.expiration.UnixNano() <= time.Now().UnixNano() {
				dataToClear = append(dataToClear, key)
			}
		}

		for _, key := range dataToClear {
			delete(c.data, key)
		}
	default:
	}
}

func (c *Cache) clear(key string) {

}

func (c *Cache) SetWithoutExp(key string, value string) error {
	return c.SetWithExp(key, value, c.defaultExp)
}

func (c *Cache) SetWithExp(key string, value string, exp time.Duration) error {
	expiration := time.Now().Add(exp)
	c.mu.RLock()
	_, ok := c.data[key]
	c.mu.RUnlock()
	if ok {
		return errors.New("key already reserved")
	}
	c.mu.Lock()
	c.data[key] = &cacheItem{
		val:        value,
		expiration: expiration,
	}
	c.mu.Unlock()
	return nil
}

func (c *Cache) Get(key string) (string, error) {
	c.mu.RLock()
	val, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		return "", errors.New("no such key")
	}
	if val.expiration.UnixNano() <= time.Now().UnixNano() {
		go func(key string, val *cacheItem, c *Cache) {
			c.mu.Lock()
			defer c.mu.Unlock()
			delete(c.data, key)
		}(key, val, c)
		return "", errors.New("already deleted")
	}

	return val.val.(string), nil
}

func (c *Cache) RSet(key, value string) int {
	var l int
	var data []string
	c.mu.RLock()
	v, ok := c.data[key]
	if ok {
		data = v.val.([]string)
		l = len(data)
	}
	c.mu.RUnlock()
	c.mu.Lock()
	data = append(data, value)
	if !ok {
		c.data[key] = &cacheItem{
			expiration: time.Now().Add(c.defaultExp),
		}
	}
	c.data[key].val = data
	c.mu.Unlock()
	return l + 1
}
