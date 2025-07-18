package datasource

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PythonicVarun/Stratum/internal/config"
	"github.com/PythonicVarun/Stratum/internal/database"
)

// DataSource defines the interface for any data source (DB, API, etc.).
type DataSource interface {
	Fetch(idValue string) ([]byte, error)
}

// Factory function that returns the correct data source based on the project's configuration.
func NewDataSource(p config.Project, dbManager *database.ConnectionManager, config *config.AppConfig) (DataSource, error) {
	switch p.SourceType {
	case "database":
		db, err := dbManager.Get(p.DB_DSN)
		if err != nil {
			return nil, fmt.Errorf("failed to get DB connection: %w", err)
		}
		return &DatabaseSource{
			db:      db,
			project: p,
			config:  config,
		}, nil
	case "api":
		return &APISource{
			project: p,
			client:  &http.Client{},
			config:  config,
		}, nil
	default:
		return nil, fmt.Errorf("unknown source type: %s", p.SourceType)
	}
}

type DatabaseSource struct {
	db      database.DBLoader
	project config.Project
	config  *config.AppConfig
}

func (s *DatabaseSource) Fetch(idValue string) ([]byte, error) {
	data, err := s.db.Fetch(s.project.Table, s.project.IdColumn, s.project.ServeColumn, idValue)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	content := string(data)

	if strings.HasPrefix(content, "data:") {
		parts := strings.SplitN(content, ",", 2)
		if len(parts) == 2 && strings.Contains(parts[0], ";base64") {
			base64Data := parts[1]
			decodedData, err := base64.StdEncoding.DecodeString(base64Data)
			if err == nil {
				return decodedData, nil
			}
		}
	}

	if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
		req, err := http.NewRequest("GET", content, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for URL %s: %w", content, err)
		}

		if s.config.ApiClientUserAgent != "" {
			req.Header.Set("User-Agent", s.config.ApiClientUserAgent)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch data from URL %s: %w", content, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusNotFound {
				return nil, nil
			}
			return nil, fmt.Errorf("URL fetch returned non-200 status: %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	}

	decodedData, err := base64.StdEncoding.DecodeString(content)
	if err == nil {
		return decodedData, nil
	}

	return data, nil
}

type APISource struct {
	project config.Project
	client  *http.Client
	config  *config.AppConfig
}

func (s *APISource) Fetch(idValue string) ([]byte, error) {
	targetURL := strings.Replace(s.project.APIEndpoint, "{"+s.project.IdColumn+"}", idValue, 1)

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create API request: %w", err)
	}

	if s.config.ApiClientUserAgent != "" {
		req.Header.Add("User-Agent", s.config.ApiClientUserAgent)
	}

	// Add authorization headers based on the project's config
	switch s.project.APIAuthType {
	case "bearer":
		authHeader := fmt.Sprintf("Bearer %s", s.project.APIAuthSecret)
		req.Header.Add("Authorization", authHeader)
	case "header":
		req.Header.Add(s.project.APIAuthHeaderName, s.project.APIAuthSecret)
	case "none":
		// No auth header needed
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute API request to %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("API request to %s returned non-200 status: %s", targetURL, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response body: %w", err)
	}

	return body, nil
}
