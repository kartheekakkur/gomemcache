package gomemcache

import (
	"testing"
	"time"
)

func TestCacheIntialization(t *testing.T) {
	cache := NewCache()

	if cache == nil {
		t.Errorf("NewCache() =%v, want non-nil", cache)
	}
}

func TestCacheSetAndGetBehaviour(t *testing.T) {
	cache := NewCache()
	cache.Set("key1", "value1", time.Minute)
	value, found := cache.Get("key1")

	if !found || value != "value1" {
		t.Errorf("Get() = %v,%v want %v,%v", value, found, "value1", true)
	}
}
