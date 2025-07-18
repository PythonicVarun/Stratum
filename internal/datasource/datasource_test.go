package datasource

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PythonicVarun/Stratum/internal/config"
	"github.com/stretchr/testify/assert"
)

// mockDBLoader allows faking the database fetch behavior.
type mockDBLoader struct {
	FetchFunc func(table, idColumn, serveColumn, idValue string) ([]byte, error)
}

func (m *mockDBLoader) Fetch(table, idColumn, serveColumn, idValue string) ([]byte, error) {
	if m.FetchFunc != nil {
		return m.FetchFunc(table, idColumn, serveColumn, idValue)
	}
	return nil, errors.New("FetchFunc not implemented")
}

func (m *mockDBLoader) Close() {}

func TestDatabaseSource_Fetch(t *testing.T) {
	t.Run("Direct Data", func(t *testing.T) {
		mockDB := &mockDBLoader{
			FetchFunc: func(table, idColumn, serveColumn, idValue string) ([]byte, error) {
				return []byte("direct_data"), nil
			},
		}
		ds := &DatabaseSource{db: mockDB}
		data, err := ds.Fetch("1")
		assert.NoError(t, err)
		assert.Equal(t, []byte("direct_data"), data)
	})

	t.Run("Base64 Data", func(t *testing.T) {
		mockDB := &mockDBLoader{
			FetchFunc: func(table, idColumn, serveColumn, idValue string) ([]byte, error) {
				// "test" -> "dGVzdA=="
				return []byte("dGVzdA=="), nil
			},
		}
		ds := &DatabaseSource{db: mockDB}
		data, err := ds.Fetch("1")
		assert.NoError(t, err)
		assert.Equal(t, []byte("test"), data)
	})

	t.Run("Data URI", func(t *testing.T) {
		mockDB := &mockDBLoader{
			FetchFunc: func(table, idColumn, serveColumn, idValue string) ([]byte, error) {
				// "test" -> "dGVzdA=="
				return []byte("data:text/plain;base64,dGVzdA=="), nil
			},
		}
		ds := &DatabaseSource{db: mockDB}
		data, err := ds.Fetch("1")
		assert.NoError(t, err)
		assert.Equal(t, []byte("test"), data)
	})

	t.Run("HTTP URL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("http_data"))
		}))
		defer server.Close()

		mockDB := &mockDBLoader{
			FetchFunc: func(table, idColumn, serveColumn, idValue string) ([]byte, error) {
				return []byte(server.URL), nil
			},
		}
		ds := &DatabaseSource{db: mockDB, config: &config.AppConfig{}}
		data, err := ds.Fetch("1")
		assert.NoError(t, err)
		assert.Equal(t, []byte("http_data"), data)
	})

	t.Run("Fetch Error", func(t *testing.T) {
		mockDB := &mockDBLoader{
			FetchFunc: func(table, idColumn, serveColumn, idValue string) ([]byte, error) {
				return nil, errors.New("db error")
			},
		}
		ds := &DatabaseSource{db: mockDB}
		_, err := ds.Fetch("1")
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})
}

func TestAPISource_Fetch(t *testing.T) {
	t.Run("Successful Fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-agent", r.Header.Get("User-Agent"))
			w.Write([]byte("api_data"))
		}))
		defer server.Close()

		p := config.Project{APIEndpoint: server.URL, IdColumn: "id"}
		cfg := &config.AppConfig{ApiClientUserAgent: "test-agent"}
		ds := &APISource{project: p, client: server.Client(), config: cfg}

		data, err := ds.Fetch("1")
		assert.NoError(t, err)
		assert.Equal(t, []byte("api_data"), data)
	})

	t.Run("Bearer Auth", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer my-token", r.Header.Get("Authorization"))
			w.Write([]byte("authed_data"))
		}))
		defer server.Close()

		p := config.Project{
			APIEndpoint:   server.URL,
			IdColumn:      "id",
			APIAuthType:   "bearer",
			APIAuthSecret: "my-token",
		}
		ds := &APISource{project: p, client: server.Client(), config: &config.AppConfig{}}

		data, err := ds.Fetch("1")
		assert.NoError(t, err)
		assert.Equal(t, []byte("authed_data"), data)
	})

	t.Run("Header Auth", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "my-secret", r.Header.Get("X-API-Key"))
			w.Write([]byte("header_authed_data"))
		}))
		defer server.Close()

		p := config.Project{
			APIEndpoint:       server.URL,
			IdColumn:          "id",
			APIAuthType:       "header",
			APIAuthHeaderName: "X-API-Key",
			APIAuthSecret:     "my-secret",
		}
		ds := &APISource{project: p, client: server.Client(), config: &config.AppConfig{}}

		data, err := ds.Fetch("1")
		assert.NoError(t, err)
		assert.Equal(t, []byte("header_authed_data"), data)
	})

	t.Run("Not Found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		p := config.Project{APIEndpoint: server.URL, IdColumn: "id"}
		ds := &APISource{project: p, client: server.Client(), config: &config.AppConfig{}}

		data, err := ds.Fetch("1")
		assert.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("Server Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		p := config.Project{APIEndpoint: server.URL, IdColumn: "id"}
		ds := &APISource{project: p, client: server.Client(), config: &config.AppConfig{}}

		_, err := ds.Fetch("1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-200 status")
	})
}
