package main

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type cacheEntry[T any] struct {
	value     T
	timestamp time.Time
}

type Cache[T any] struct {
	logger     *slog.Logger
	mu         sync.RWMutex
	cache      map[string]cacheEntry[T]
	name       string
	expiration time.Duration
}

func New[T any](ctx context.Context, logger *slog.Logger, name string, expiration time.Duration) *Cache[T] {
	c := Cache[T]{
		cache:      make(map[string]cacheEntry[T]),
		logger:     logger,
		name:       name,
		expiration: expiration,
	}

	// start invalidator go function
	go c.invalidator(ctx)

	return &c
}

func (c *Cache[T]) invalidator(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			expiration := time.Now().Add(c.expiration * -1)
			for k, v := range c.cache {
				if v.timestamp.Before(expiration) {
					c.logger.Debug("deleting expired cache entry", slog.String("name", c.name), slog.String("key", k), slog.Time("timestamp", v.timestamp), slog.Time("expiration", expiration))
					delete(c.cache, k)
				}
			}
			c.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if x, ok := c.cache[key]; ok {
		c.logger.Debug("returning cached entry", slog.String("name", c.name), slog.String("key", key), slog.Time("timestamp", x.timestamp))
		return x.value, true
	}
	var result T
	return result, false
}

func (c *Cache[T]) Set(key string, value T) {
	c.mu.Lock()
	timestamp := time.Now()
	c.logger.Debug("setting cache entry", slog.String("name", c.name), slog.String("key", key), slog.Time("timestamp", timestamp))
	c.cache[key] = cacheEntry[T]{
		value:     value,
		timestamp: timestamp,
	}
	c.mu.Unlock()
}
