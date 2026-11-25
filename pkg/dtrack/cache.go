package dtrack

import (
	"container/list"
	"context"
	"net/http"
	"sync"
	"time"
)

type noCacheKey struct{}

// WithoutCache makes requests made with ctx bypass the response cache, e.g. for health checks.
func WithoutCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, noCacheKey{}, true)
}

func cacheDisabled(ctx context.Context) bool {
	v, _ := ctx.Value(noCacheKey{}).(bool)
	return v
}

type cachedResponse struct {
	key     string
	body    []byte
	header  http.Header
	expires time.Time
}

// responseCache is a byte-bounded LRU of successful GET responses.
type responseCache struct {
	mu       sync.Mutex
	ttl      time.Duration
	maxBytes int
	size     int
	order    *list.List
	entries  map[string]*list.Element
	now      func() time.Time
}

func newResponseCache(ttl time.Duration, maxBytes int) *responseCache {
	return &responseCache{
		ttl:      ttl,
		maxBytes: maxBytes,
		order:    list.New(),
		entries:  map[string]*list.Element{},
		now:      time.Now,
	}
}

func (c *responseCache) get(key string) (*cachedResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	entry := el.Value.(*cachedResponse)
	if c.now().After(entry.expires) {
		c.remove(el)
		return nil, false
	}
	c.order.MoveToFront(el)
	return entry, true
}

func (c *responseCache) set(key string, body []byte, header http.Header) {
	if len(body) > c.maxBytes {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[key]; ok {
		c.remove(el)
	}
	entry := &cachedResponse{key: key, body: body, header: header, expires: c.now().Add(c.ttl)}
	c.entries[key] = c.order.PushFront(entry)
	c.size += len(body)
	for c.size > c.maxBytes {
		c.remove(c.order.Back())
	}
}

func (c *responseCache) remove(el *list.Element) {
	entry := c.order.Remove(el).(*cachedResponse)
	delete(c.entries, entry.key)
	c.size -= len(entry.body)
}
