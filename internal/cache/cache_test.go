package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, string) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub redis connection", err)
	}
	return s, s.Addr()
}

func TestNewRedisCache(t *testing.T) {
	s, addr := setupMiniredis(t)
	defer s.Close()

	t.Run("Valid Redis URL", func(t *testing.T) {
		cache, err := NewRedisCache("redis://" + addr)
		assert.NoError(t, err)
		assert.NotNil(t, cache)
		defer cache.Close()
	})

	t.Run("Invalid Redis URL", func(t *testing.T) {
		_, err := NewRedisCache("redis://invalid-url:port")
		assert.Error(t, err)
	})

	t.Run("Empty Redis URL", func(t *testing.T) {
		_, err := NewRedisCache("")
		assert.Error(t, err)
		assert.Equal(t, "redis URL is not provided", err.Error())
	})
}

func TestRedisCache_GetSet(t *testing.T) {
	s, addr := setupMiniredis(t)
	defer s.Close()

	cache, err := NewRedisCache("redis://" + addr)
	assert.NoError(t, err)
	defer cache.Close()

	ctx := context.Background()
	key := "test_key"
	value := []byte("test_value")

	// Test Set
	err = cache.Set(ctx, key, value, 1*time.Minute)
	assert.NoError(t, err)

	// Test Get
	retrievedValue, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrievedValue)

	// Test Get on non-existent key
	retrievedValue, err = cache.Get(ctx, "non_existent_key")
	assert.NoError(t, err)
	assert.Nil(t, retrievedValue)
}

func TestRedisCache_TTL(t *testing.T) {
	s, addr := setupMiniredis(t)
	defer s.Close()

	cache, err := NewRedisCache("redis://" + addr)
	assert.NoError(t, err)
	defer cache.Close()

	ctx := context.Background()
	key := "ttl_key"
	value := []byte("ttl_value")

	// Set with a short TTL
	err = cache.Set(ctx, key, value, 1*time.Second)
	assert.NoError(t, err)

	// Check if key exists
	retrievedValue, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrievedValue)

	// Wait for TTL to expire
	s.FastForward(2 * time.Second)

	// Check if key has expired
	retrievedValue, err = cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.Nil(t, retrievedValue)
}

func TestRedisCache_Close(t *testing.T) {
	s, addr := setupMiniredis(t)
	defer s.Close()

	cache, err := NewRedisCache("redis://" + addr)
	assert.NoError(t, err)

	err = cache.Close()
	assert.NoError(t, err)
}
