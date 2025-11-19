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
	val        string
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
	go c.clear()
}

func (c *Cache) clear() {
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

func (c *Cache) SetWithoutExp(key string, value string) error {
	return c.SetWithExp(key, value, c.defaultExp)
}

func (c *Cache) SetWithExp(key string, value string, exp time.Duration) error {
	expiration := time.Now().Add(exp)
	c.mu.RLock()
	_, ok := c.data[key]
	if ok {
		return errors.New("key already reserved")
	}
	c.mu.RUnlock()
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

	return val.val, nil
}
