package datasource

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Cache provides simple file-based caching for API responses
type Cache struct {
	cacheDir string
	ttl      time.Duration
	mu       sync.RWMutex
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Key       string    `json:"key"`
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// NewCache creates a new cache instance
func NewCache(cacheDir string, ttl time.Duration) *Cache {
	if cacheDir == "" {
		cacheDir = "cache/forensic"
	}

	// Create cache directory if it doesn't exist
	os.MkdirAll(cacheDir, 0755)

	return &Cache{
		cacheDir: cacheDir,
		ttl:      ttl,
	}
}

// Get retrieves an item from cache
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cacheFile := c.getCacheFilePath(key)

	// Check if file exists
	info, err := os.Stat(cacheFile)
	if err != nil {
		return nil, false
	}

	// Check if cache is expired
	if time.Since(info.ModTime()) > c.ttl {
		// Cache expired, delete it
		os.Remove(cacheFile)
		return nil, false
	}

	// Read cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	return entry.Data, true
}

// Set stores an item in cache
func (c *Cache) Set(key string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := CacheEntry{
		Key:       key,
		Data:      data,
		Timestamp: time.Now(),
	}

	entryData, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	cacheFile := c.getCacheFilePath(key)
	return os.WriteFile(cacheFile, entryData, 0644)
}

// Delete removes an item from cache
func (c *Cache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cacheFile := c.getCacheFilePath(key)
	return os.Remove(cacheFile)
}

// Clear removes all cache entries
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return os.RemoveAll(c.cacheDir)
}

// CleanupExpired removes expired cache entries
func (c *Cache) CleanupExpired() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if time.Since(info.ModTime()) > c.ttl {
			os.Remove(filepath.Join(c.cacheDir, entry.Name()))
		}
	}

	return nil
}

func (c *Cache) getCacheFilePath(key string) string {
	// Create MD5 hash of key for filename
	hash := md5.Sum([]byte(key))
	filename := fmt.Sprintf("%x.json", hash)
	return filepath.Join(c.cacheDir, filename)
}

// GetOrFetch retrieves from cache or fetches using provided function
func (c *Cache) GetOrFetch(key string, fetchFn func() ([]byte, error)) ([]byte, error) {
	// Try to get from cache first
	if data, ok := c.Get(key); ok {
		return data, nil
	}

	// Fetch fresh data
	data, err := fetchFn()
	if err != nil {
		return nil, err
	}

	// Store in cache (ignore errors)
	c.Set(key, data)

	return data, nil
}

// MakeKey creates a cache key from parts
func MakeKey(parts ...string) string {
	return fmt.Sprintf("%s", parts)
}
