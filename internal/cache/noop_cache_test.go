package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNoOpCache(t *testing.T) {
	cache := &NoOpCache{}
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		val, err := cache.Get(ctx, "any_key")
		assert.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("Set", func(t *testing.T) {
		err := cache.Set(ctx, "any_key", []byte("any_value"), 1*time.Minute)
		assert.NoError(t, err)
	})

	t.Run("Close", func(t *testing.T) {
		err := cache.Close()
		assert.NoError(t, err)
	})
}
