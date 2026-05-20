package marketplace

import (
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	c := NewCache()
	if c == nil {
		t.Fatal("NewCache returned nil")
	}
	if c.ttl != 10*time.Minute {
		t.Errorf("ttl = %v, want 10m", c.ttl)
	}
}

func TestCache_SetAndGet(t *testing.T) {
	c := NewCache()
	c.Set("key1", "value1")

	got := c.Get("key1")
	if got != "value1" {
		t.Errorf("Get(key1) = %v, want value1", got)
	}
}

func TestCache_GetMissing(t *testing.T) {
	c := NewCache()
	got := c.Get("nonexistent")
	if got != nil {
		t.Errorf("Get(nonexistent) = %v, want nil", got)
	}
}

func TestCache_Invalidate(t *testing.T) {
	c := NewCache()
	c.Set("key1", "value1")
	c.Invalidate("key1")

	got := c.Get("key1")
	if got != nil {
		t.Errorf("after Invalidate, Get = %v, want nil", got)
	}
}

func TestCache_ExpiredItem(t *testing.T) {
	c := &Cache{
		items: make(map[string]*cacheItem),
		ttl:   1 * time.Millisecond,
	}
	c.Set("key1", "value1")

	time.Sleep(5 * time.Millisecond)
	got := c.Get("key1")
	if got != nil {
		t.Errorf("expired item should return nil, got %v", got)
	}
}

func TestCache_OverwriteKey(t *testing.T) {
	c := NewCache()
	c.Set("key", "v1")
	c.Set("key", "v2")

	got := c.Get("key")
	if got != "v2" {
		t.Errorf("Get = %v, want v2", got)
	}
}

func TestCache_DifferentTypes(t *testing.T) {
	c := NewCache()
	c.Set("int", 42)
	c.Set("slice", []string{"a", "b"})
	c.Set("struct", &Registry{UpdatedAt: "now"})

	if c.Get("int") != 42 {
		t.Error("int value mismatch")
	}
	sl, ok := c.Get("slice").([]string)
	if !ok || len(sl) != 2 {
		t.Error("slice value mismatch")
	}
	reg, ok := c.Get("struct").(*Registry)
	if !ok || reg.UpdatedAt != "now" {
		t.Error("struct value mismatch")
	}
}
