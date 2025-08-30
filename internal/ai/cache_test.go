package ai

import (
	"fmt"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	ttl := 5 * time.Minute
	cache := NewCache(100, ttl)

	if cache == nil {
		t.Fatal("NewCache() returned nil")
	}

	if cache.ttl != ttl {
		t.Errorf("Expected TTL %v, got %v", ttl, cache.ttl)
	}

	if cache.items == nil {
		t.Error("Cache items map is nil")
	}
}

func TestCacheSetAndGet(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)

	// Test setting and getting a value
	key := "test_key"
	value := "test_value"

	cache.Set(key, value)

	got := cache.Get(key)
	if got == nil {
		t.Error("Expected to find cached value")
	}

	if got != value {
		t.Errorf("Expected %v, got %v", value, got)
	}
}

func TestCacheGetNonExistent(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)

	// Test getting a non-existent key
	got := cache.Get("non_existent_key")
	if got != nil {
		t.Errorf("Expected nil value, got %v", got)
	}
}

func TestCacheDelete(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)

	key := "test_key"
	value := "test_value"

	// Set a value
	cache.Set(key, value)

	// Verify it exists
	got := cache.Get(key)
	if got == nil {
		t.Error("Expected to find cached value before deletion")
	}

	// Delete the value
	cache.Delete(key)

	// Verify it's gone
	got = cache.Get(key)
	if got != nil {
		t.Errorf("Expected nil value after deletion, got %v", got)
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)

	// Set multiple values
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Verify they exist
	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}

	// Clear the cache
	cache.Clear()

	// Verify cache is empty
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}

	// Verify values are gone
	got := cache.Get("key1")
	if got != nil {
		t.Errorf("Expected nil value after clear, got %v", got)
	}
}

func TestCacheSize(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)

	// Initially empty
	if cache.Size() != 0 {
		t.Errorf("Expected size 0, got %d", cache.Size())
	}

	// Add some items
	cache.Set("key1", "value1")
	if cache.Size() != 1 {
		t.Errorf("Expected size 1, got %d", cache.Size())
	}

	cache.Set("key2", "value2")
	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}

	// Delete an item
	cache.Delete("key1")
	if cache.Size() != 1 {
		t.Errorf("Expected size 1 after deletion, got %d", cache.Size())
	}

	// Clear all items
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := NewCache(100, 100*time.Millisecond) // Very short TTL for testing
	key := "test_key"
	value := "test_value"

	// Set a value
	cache.Set(key, value)

	// Verify it exists immediately
	got := cache.Get(key)
	if got == nil {
		t.Error("Expected to find cached value immediately after setting")
	}
	if got != value {
		t.Errorf("Expected %v, got %v", value, got)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Verify it's expired
	got = cache.Get(key)
	if got != nil {
		t.Errorf("Expected nil value for expired key, got %v", got)
	}
}

func TestCacheItemIsExpired(t *testing.T) {
	// Test with expired item
	expiredItem := &CacheItem{
		Value:     "test",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // 1 hour ago
	}

	if !expiredItem.IsExpired() {
		t.Error("Expected item to be expired")
	}

	// Test with non-expired item
	validItem := &CacheItem{
		Value:     "test",
		ExpiresAt: time.Now().Add(1 * time.Hour), // 1 hour from now
	}

	if validItem.IsExpired() {
		t.Error("Expected item not to be expired")
	}
}

func TestCacheOverwrite(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)

	key := "test_key"
	value1 := "value1"
	value2 := "value2"

	// Set initial value
	cache.Set(key, value1)
	got := cache.Get(key)
	if got == nil || got != value1 {
		t.Errorf("Expected %v, got %v", value1, got)
	}

	// Overwrite with new value
	cache.Set(key, value2)
	got = cache.Get(key)
	if got == nil || got != value2 {
		t.Errorf("Expected %v, got %v", value2, got)
	}

	// Size should still be 1
	if cache.Size() != 1 {
		t.Errorf("Expected size 1, got %d", cache.Size())
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)

	// Test concurrent writes and reads
	done := make(chan bool, 2)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key_%d", i)
			value := fmt.Sprintf("value_%d", i)
			cache.Set(key, value)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key_%d", i)
			cache.Get(key) // Don't care about the result, just testing for race conditions
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// If we get here without a race condition, the test passes
}
