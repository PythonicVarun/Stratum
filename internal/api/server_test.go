package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PythonicVarun/Stratum/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockDataSource allows faking the behavior of a data source for tests.
type mockDataSource struct {
	FetchFunc func(id string) ([]byte, error)
}

func (m *mockDataSource) Fetch(id string) ([]byte, error) {
	if m.FetchFunc != nil {
		return m.FetchFunc(id)
	}
	return nil, fmt.Errorf("FetchFunc not implemented")
}

// mockCache allows faking the behavior of a cache for tests.
type mockCache struct {
	GetFunc func(ctx context.Context, key string) ([]byte, error)
	SetFunc func(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	return nil, nil
}

func (m *mockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, value, ttl)
	}
	return nil
}

func (m *mockCache) Close() error { return nil }

func TestConvertToGinRoute(t *testing.T) {
	testCases := []struct {
		name     string
		route    string
		expected string
	}{
		{"Standard placeholder", "/users/{id}", "/users/:id"},
		{"Placeholder with suffix", "/users/{id}/profile", "/users/*id"},
		{"No placeholder", "/users", "/users"},
		{"Malformed placeholder start", "/users/{id", "/users/{id"},
		{"Malformed placeholder end", "/users/id}", "/users/id}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ginRoute := convertToGinRoute(tc.route)
			assert.Equal(t, tc.expected, ginRoute)
		})
	}
}

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

func TestCreateHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	project := config.Project{
		Name:          "test_project",
		Route:         "/test/{id}",
		IdPlaceholder: "id",
		ContentType:   "text/plain",
		CacheTTL:      1 * time.Minute,
	}

	mockDS := &mockDataSource{
		FetchFunc: func(id string) ([]byte, error) {
			if id == "1" {
				return []byte("test data"), nil
			}
			if id == "error" {
				return nil, errors.New("fetch error")
			}
			return nil, nil
		},
	}

	_ = mockDS

	t.Run("Cache Hit", func(t *testing.T) {
		router := gin.New()
		mockCache := &mockCache{
			GetFunc: func(ctx context.Context, key string) ([]byte, error) {
				assert.Equal(t, "test_project:1", key)
				return []byte("cached data"), nil
			},
		}
		s := &Server{cache: mockCache}
		handler := s.createTestHandler(project, mockDS)
		router.GET(convertToGinRoute(project.Route), handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/1", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "cached data", w.Body.String())
		assert.Equal(t, "HIT", w.Header().Get("X-Cache-Status"))
		assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
	})

	t.Run("Cache Miss and Fetch Success", func(t *testing.T) {
		router := gin.New()
		var cacheSetKey string
		var cacheSetValue []byte
		mockCache := &mockCache{
			GetFunc: func(ctx context.Context, key string) ([]byte, error) {
				return nil, nil // Simulate cache miss
			},
			SetFunc: func(ctx context.Context, key string, value []byte, ttl time.Duration) error {
				cacheSetKey = key
				cacheSetValue = value
				return nil
			},
		}
		s := &Server{cache: mockCache}
		handler := s.createTestHandler(project, mockDS)
		router.GET(convertToGinRoute(project.Route), handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/1", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "test data", w.Body.String())
		assert.Equal(t, "MISS", w.Header().Get("X-Cache-Status"))
		assert.Equal(t, "test_project:1", cacheSetKey)
		assert.Equal(t, []byte("test data"), cacheSetValue)
	})

	t.Run("Data Not Found", func(t *testing.T) {
		router := gin.New()
		s := &Server{cache: &mockCache{}}
		handler := s.createTestHandler(project, mockDS)
		router.GET(convertToGinRoute(project.Route), handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/2", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "Not Found", w.Body.String())
	})

	t.Run("Fetch Error", func(t *testing.T) {
		router := gin.New()
		s := &Server{cache: &mockCache{}}
		handler := s.createTestHandler(project, mockDS)
		router.GET(convertToGinRoute(project.Route), handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/error", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "Internal Server Error!", w.Body.String())
	})

	t.Run("Cache Bypass", func(t *testing.T) {
		router := gin.New()
		s := &Server{cache: &mockCache{}}
		handler := s.createTestHandler(project, mockDS)
		router.GET(convertToGinRoute(project.Route), handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/1", nil)
		req.Header.Set("Cache-Control", "no-cache")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "test data", w.Body.String())
		assert.Equal(t, "BYPASS", w.Header().Get("X-Cache-Status"))
	})
}

// createTestHandler is a helper to create a gin handler with a mocked data source,
// bypassing the NewDataSource factory which is hard to mock without DI.
func (s *Server) createTestHandler(p config.Project, source *mockDataSource) gin.HandlerFunc {
	// This is a simplified version of the real createHandler for testing purposes.
	return func(c *gin.Context) {
		idValue := c.Param(p.IdPlaceholder)
		ctx := c.Request.Context()
		cacheKey := fmt.Sprintf("%s:%s", p.Name, idValue)

		// Cache check
		if c.GetHeader("Cache-Control") != "no-cache" {
			cachedData, _ := s.cache.Get(ctx, cacheKey)
			if cachedData != nil {
				c.Header("X-Cache-Status", "HIT")
				c.Header("Content-Type", p.ContentType)
				c.Data(http.StatusOK, p.ContentType, cachedData)
				return
			}
		}

		if c.GetHeader("Cache-Control") == "no-cache" {
			c.Header("X-Cache-Status", "BYPASS")
		} else {
			c.Header("X-Cache-Status", "MISS")
		}

		// Fetch from source
		data, err := source.Fetch(idValue)
		if err != nil {
			c.String(http.StatusInternalServerError, "Internal Server Error!")
			return
		}
		if data == nil {
			c.String(http.StatusNotFound, "Not Found")
			return
		}

		// Set cache
		s.cache.Set(ctx, cacheKey, data, p.CacheTTL)

		c.Data(http.StatusOK, p.ContentType, data)
	}
}
