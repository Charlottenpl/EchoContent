package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/charlottepl/blog-system/internal/core/config"
)

// Cache 缓存接口
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Flush(ctx context.Context) error
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	store  sync.Map
	closed bool
	mu     sync.RWMutex
}

// CacheItem 缓存项
type CacheItem struct {
	Value      interface{}
	Expiration time.Time
}

// NewMemoryCache 创建内存缓存实例
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{}

	// 启动清理协程
	go cache.startCleanup()

	return cache
}

// Get 获取缓存值
func (c *MemoryCache) Get(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return "", fmt.Errorf("cache is closed")
	}

	if item, ok := c.store.Load(key); ok {
		cacheItem := item.(*CacheItem)
		if !cacheItem.Expiration.IsZero() && time.Now().After(cacheItem.Expiration) {
			c.store.Delete(key)
			return "", fmt.Errorf("cache expired")
		}

		if str, ok := cacheItem.Value.(string); ok {
			return str, nil
		}

		// 尝试序列化为JSON
		if bytes, err := json.Marshal(cacheItem.Value); err == nil {
			return string(bytes), nil
		}

		return "", fmt.Errorf("failed to serialize cache value")
	}

	return "", fmt.Errorf("key not found")
}

// Set 设置缓存值
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("cache is closed")
	}

	var expTime time.Time
	if expiration > 0 {
		expTime = time.Now().Add(expiration)
	}

	c.store.Store(key, &CacheItem{
		Value:      value,
		Expiration: expTime,
	})

	return nil
}

// Delete 删除缓存值
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("cache is closed")
	}

	c.store.Delete(key)
	return nil
}

// Exists 检查键是否存在
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return false, fmt.Errorf("cache is closed")
	}

	if item, ok := c.store.Load(key); ok {
		cacheItem := item.(*CacheItem)
		if !cacheItem.Expiration.IsZero() && time.Now().After(cacheItem.Expiration) {
			c.store.Delete(key)
			return false, nil
		}
		return true, nil
	}

	return false, nil
}

// Flush 清空所有缓存
func (c *MemoryCache) Flush(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("cache is closed")
	}

	c.store = sync.Map{}
	return nil
}

// Close 关闭缓存
func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.store = sync.Map{}
	return nil
}

// startCleanup 启动清理过期缓存的协程
func (c *MemoryCache) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup 清理过期的缓存项
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	now := time.Now()
	c.store.Range(func(key, value interface{}) bool {
		cacheItem := value.(*CacheItem)
		if !cacheItem.Expiration.IsZero() && now.After(cacheItem.Expiration) {
			c.store.Delete(key)
		}
		return true
	})
}

// 全局缓存实例
var GlobalCache Cache

// Init 初始化缓存系统
func Init() error {
	cfg := config.Get()
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	switch cfg.Cache.Type {
	case "memory":
		GlobalCache = NewMemoryCache()
	default:
		GlobalCache = NewMemoryCache()
	}

	return nil
}

// GetCache 获取全局缓存实例
func GetCache() Cache {
	return GlobalCache
}

// Get 获取缓存值
func Get(ctx context.Context, key string) (string, error) {
	if GlobalCache == nil {
		return "", fmt.Errorf("cache not initialized")
	}
	return GlobalCache.Get(ctx, key)
}

// Set 设置缓存值
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if GlobalCache == nil {
		return fmt.Errorf("cache not initialized")
	}
	return GlobalCache.Set(ctx, key, value, expiration)
}

// Delete 删除缓存值
func Delete(ctx context.Context, key string) error {
	if GlobalCache == nil {
		return fmt.Errorf("cache not initialized")
	}
	return GlobalCache.Delete(ctx, key)
}

// Exists 检查键是否存在
func Exists(ctx context.Context, key string) (bool, error) {
	if GlobalCache == nil {
		return false, fmt.Errorf("cache not initialized")
	}
	return GlobalCache.Exists(ctx, key)
}

// Flush 清空所有缓存
func Flush(ctx context.Context) error {
	if GlobalCache == nil {
		return fmt.Errorf("cache not initialized")
	}
	return GlobalCache.Flush(ctx)
}

// Close 关闭缓存
func Close() error {
	if GlobalCache != nil {
		if closer, ok := GlobalCache.(*MemoryCache); ok {
			return closer.Close()
		}
	}
	return nil
}