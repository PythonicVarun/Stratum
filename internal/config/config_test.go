package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExtractIDPlaceholder(t *testing.T) {
	testCases := []struct {
		name          string
		route         string
		expectedID    string
		expectError   bool
		expectedError string
	}{
		{
			name:        "Valid route",
			route:       "/api/users/{user_id}/avatar",
			expectedID:  "user_id",
			expectError: false,
		},
		{
			name:          "No placeholder",
			route:         "/api/users/avatar",
			expectError:   true,
			expectedError: "no '{' found in route",
		},
		{
			name:          "Malformed placeholder missing closing brace",
			route:         "/api/users/{user_id/avatar",
			expectError:   true,
			expectedError: "no '}' found after '{' in route",
		},
		{
			name:          "Malformed placeholder missing opening brace",
			route:         "/api/users/user_id}/avatar",
			expectError:   true,
			expectedError: "no '{' found in route",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := extractIDPlaceholder(tc.route)

			if tc.expectError {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, id)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Helper to set env vars and cleanup after test
	setenv := func(t *testing.T, key, value string) {
		t.Helper()
		originalValue := os.Getenv(key)
		os.Setenv(key, value)
		t.Cleanup(func() {
			os.Setenv(key, originalValue)
		})
	}

	// Cleanup all project-related env vars before and after tests
	cleanupEnv := func() {
		for i := 1; i < 10; i++ { // Assuming max 9 projects for testing
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_ROUTE", i))
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_ID_COLUMN", i))
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_CONTENT_TYPE", i))
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_CACHE_TTL_SECONDS", i))
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_SOURCE_TYPE", i))
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_DB_DSN", i))
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_TABLE", i))
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_SERVE_COLUMN", i))
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_API_ENDPOINT", i))
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_API_AUTH_TYPE", i))
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_API_AUTH_SECRET", i))
			os.Unsetenv(fmt.Sprintf("PROJECT_%d_API_AUTH_HEADER_NAME", i))
		}
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("API_CLIENT_USER_AGENT")
	}

	t.Run("Valid Database Project", func(t *testing.T) {
		cleanupEnv()
		setenv(t, "PROJECT_1_ROUTE", "/users/{id}")
		setenv(t, "PROJECT_1_ID_COLUMN", "id")
		setenv(t, "PROJECT_1_SOURCE_TYPE", "database")
		setenv(t, "PROJECT_1_DB_DSN", "user:pass@tcp(127.0.0.1:3306)/db")
		setenv(t, "PROJECT_1_TABLE", "users")
		setenv(t, "PROJECT_1_SERVE_COLUMN", "data")
		setenv(t, "PROJECT_1_CONTENT_TYPE", "application/json")
		setenv(t, "PROJECT_1_CACHE_TTL_SECONDS", "60")

		config, err := Load()
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Len(t, config.Projects, 1)
		p := config.Projects[0]
		assert.Equal(t, "/users/{id}", p.Route)
		assert.Equal(t, "id", p.IdColumn)
		assert.Equal(t, "database", p.SourceType)
		assert.Equal(t, "user:pass@tcp(127.0.0.1:3306)/db", p.DB_DSN)
		assert.Equal(t, "users", p.Table)
		assert.Equal(t, "data", p.ServeColumn)
		assert.Equal(t, "application/json", p.ContentType)
		assert.Equal(t, 60*time.Second, p.CacheTTL)
	})

	t.Run("Valid API Project with Bearer Auth", func(t *testing.T) {
		cleanupEnv()
		setenv(t, "PROJECT_1_ROUTE", "/posts/{post_id}")
		setenv(t, "PROJECT_1_ID_COLUMN", "post_id")
		setenv(t, "PROJECT_1_SOURCE_TYPE", "api")
		setenv(t, "PROJECT_1_API_ENDPOINT", "https://example.com/api/posts")
		setenv(t, "PROJECT_1_API_AUTH_TYPE", "bearer")
		setenv(t, "PROJECT_1_API_AUTH_SECRET", "my-secret-token")

		config, err := Load()
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Len(t, config.Projects, 1)
		p := config.Projects[0]
		assert.Equal(t, "api", p.SourceType)
		assert.Equal(t, "https://example.com/api/posts", p.APIEndpoint)
		assert.Equal(t, "bearer", p.APIAuthType)
		assert.Equal(t, "my-secret-token", p.APIAuthSecret)
	})

	t.Run("Missing DB DSN", func(t *testing.T) {
		cleanupEnv()
		setenv(t, "PROJECT_1_ROUTE", "/users/{id}")
		setenv(t, "PROJECT_1_ID_COLUMN", "id")
		setenv(t, "PROJECT_1_SOURCE_TYPE", "database")
		setenv(t, "PROJECT_1_TABLE", "users")
		setenv(t, "PROJECT_1_SERVE_COLUMN", "data")

		_, err := Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required database configuration")
	})

	t.Run("Missing API Endpoint", func(t *testing.T) {
		cleanupEnv()
		setenv(t, "PROJECT_1_ROUTE", "/posts/{post_id}")
		setenv(t, "PROJECT_1_ID_COLUMN", "post_id")
		setenv(t, "PROJECT_1_SOURCE_TYPE", "api")

		_, err := Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required API configuration")
	})

	t.Run("Route placeholder mismatch", func(t *testing.T) {
		cleanupEnv()
		setenv(t, "PROJECT_1_ROUTE", "/users/{user_id}")
		setenv(t, "PROJECT_1_ID_COLUMN", "id") // Mismatch
		setenv(t, "PROJECT_1_SOURCE_TYPE", "database")
		setenv(t, "PROJECT_1_DB_DSN", "user:pass@tcp(127.0.0.1:3306)/db")
		setenv(t, "PROJECT_1_TABLE", "users")
		setenv(t, "PROJECT_1_SERVE_COLUMN", "data")

		_, err := Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "route placeholder {user_id} must match ID_COLUMN 'id'")
	})

	t.Run("No Projects", func(t *testing.T) {
		cleanupEnv()
		config, err := Load()
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Empty(t, config.Projects)
	})

	t.Run("Defaults", func(t *testing.T) {
		cleanupEnv()
		// Set minimal config for one project
		setenv(t, "PROJECT_1_ROUTE", "/users/{id}")
		setenv(t, "PROJECT_1_ID_COLUMN", "id")
		setenv(t, "PROJECT_1_DB_DSN", "user:pass@tcp(127.0.0.1:3306)/db")
		setenv(t, "PROJECT_1_TABLE", "users")
		setenv(t, "PROJECT_1_SERVE_COLUMN", "data")

		config, err := Load()
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "8080", config.ServerPort)
		assert.Equal(t, "Stratum-Server/1.0 (github.com/PythonicVarun/Stratum)", config.ApiClientUserAgent)
		assert.Len(t, config.Projects, 1)
		p := config.Projects[0]
		assert.Equal(t, "database", p.SourceType)
		assert.Equal(t, 3600*time.Second, p.CacheTTL)
	})
}
