package web

import (
	"sync"
	"time"
)

type entry struct {
	val any
	at  time.Time
}

type cache struct {
	mu       sync.Mutex
	data     map[string]entry
	inflight map[string]chan struct{}
}

func newCache() *cache {
	return &cache{
		data:     make(map[string]entry),
		inflight: make(map[string]chan struct{}),
	}
}

func (c *cache) load(key string, ttl time.Duration, fn func() (any, error)) (any, error) {
	c.mu.Lock()
	e, hit := c.data[key]
	c.mu.Unlock()

	if hit && time.Since(e.at) < ttl {
		return e.val, nil
	}
	if hit {
		go c.fetch(key, fn)
		return e.val, nil
	}
	return c.fetch(key, fn)
}

func (c *cache) fetch(key string, fn func() (any, error)) (any, error) {
	c.mu.Lock()
	if wait, running := c.inflight[key]; running {
		c.mu.Unlock()
		<-wait

		c.mu.Lock()
		e, ok := c.data[key]
		c.mu.Unlock()
		if ok {
			return e.val, nil
		}
		return fn()
	}

	done := make(chan struct{})
	c.inflight[key] = done
	c.mu.Unlock()

	val, err := fn()

	c.mu.Lock()
	if err == nil {
		c.data[key] = entry{val: val, at: time.Now()}
	}
	delete(c.inflight, key)
	c.mu.Unlock()
	close(done)

	return val, err
}

func (c *cache) invalidate(key string) {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
}
