package cache

import (
	"errors"
	"strconv"
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

func (c *Cache) RSet(key string, values ...string) int {
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
	data = append(data, values...)
	if !ok {
		c.data[key] = &cacheItem{
			expiration: time.Now().Add(c.defaultExp),
		}
	}
	c.data[key].val = data
	c.mu.Unlock()
	return l + len(values)
}

func (c *Cache) LSet(key string, values ...string) int {
	var l int
	var data []string
	c.mu.RLock()
	v, ok := c.data[key]
	if ok {
		data = v.val.([]string)
		l = len(data)
	}
	c.mu.RUnlock()
	reverseVal := make([]string, 0, len(values))
	for _, v := range values {
		reverseVal = append([]string{v}, reverseVal...)
	}
	c.mu.Lock()
	data = append(reverseVal, data...)
	if !ok {
		c.data[key] = &cacheItem{
			expiration: time.Now().Add(c.defaultExp),
		}
	}
	c.data[key].val = data
	c.mu.Unlock()
	return l + len(values)
}

func (c *Cache) RGet(key string, startStr, endStr string) ([][]byte, error) {
	start, err := strconv.Atoi(startStr)
	if err != nil {
		return nil, errors.New("start idx not int")
	}
	end, err := strconv.Atoi(endStr)
	if err != nil {
		return nil, errors.New("end idx not int")
	}

	c.mu.RLock()
	sl, ok := c.data[key]
	if !ok {
		return nil, errors.New("wrong key")
	}
	data := sl.val.([]string)
	length := len(data)
	if start >= length {
		return nil, errors.New("wrong start idx")
	}
	if start < 0 {
		start = length + start
		if start < 0 {
			start = 0
		}
	}
	if end < 0 {
		end = length + end
	}
	if start > end {
		return nil, errors.New("wrong idxes")
	}
	if end >= length {
		end = length - 1
	}
	var res [][]byte
	for i := start; i <= end; i++ {
		res = append(res, []byte(data[i]))
	}
	c.mu.RUnlock()
	return res, nil
}

func (c *Cache) GetL(key string) (int, error) {
	c.mu.RLock()
	data, ok := c.data[key]
	if !ok {
		return 0, errors.New("not created")
	}
	switch v := data.val.(type) {
	case []string:
		return len(v), nil
	default:
		return 0, errors.New("wrong type")
	}
}
