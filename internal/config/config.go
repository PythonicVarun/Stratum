package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Project struct {
	Name          string
	Route         string
	IdColumn      string
	ContentType   string
	CacheTTL      time.Duration
	IdPlaceholder string

	// Source-specific fields
	SourceType  string // "database" or "api"
	DB_DSN      string // For database source
	Table       string // For database source
	ServeColumn string // For database source
	APIEndpoint string // For api source

	// API Source Auth
	APIAuthType       string
	APIAuthSecret     string
	APIAuthHeaderName string
}

// AppConfig holds the global application configuration.
type AppConfig struct {
	Projects           []Project
	ServerPort         string
	RedisURL           string
	ApiClientUserAgent string
}

// Load scans the environment variables and builds the application configuration.
func Load() (*AppConfig, error) {
	// Google Cloud Run sets the PORT environment variable.
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("SERVER_PORT")
	}

	appConfig := &AppConfig{
		ServerPort:         port,
		RedisURL:           os.Getenv("REDIS_URL"),
		ApiClientUserAgent: os.Getenv("API_CLIENT_USER_AGENT"),
	}

	if appConfig.ServerPort == "" {
		appConfig.ServerPort = "8080" // Default port
	}

	if appConfig.ApiClientUserAgent == "" {
		appConfig.ApiClientUserAgent = "Stratum-Server/1.0 (github.com/PythonicVarun/Stratum)" // Default user agent
	}

	// Scan for projects by looking for PROJECT_{n}_ROUTE variables
	for i := 1; ; i++ {
		routeKey := fmt.Sprintf("PROJECT_%d_ROUTE", i)
		route := os.Getenv(routeKey)
		if route == "" {
			break
		}

		// Extract placeholder from route
		idPlaceholder, err := extractIDPlaceholder(route)
		if err != nil {
			return nil, fmt.Errorf("invalid route for project %d: %w", i, err)
		}

		ttlStr := os.Getenv(fmt.Sprintf("PROJECT_%d_CACHE_TTL_SECONDS", i))
		ttl, err := strconv.Atoi(ttlStr)
		if err != nil {
			ttl = 3600 // Default to 1 hour
		}

		project := Project{
			Name:          fmt.Sprintf("project_%d", i),
			Route:         route,
			IdColumn:      os.Getenv(fmt.Sprintf("PROJECT_%d_ID_COLUMN", i)),
			ContentType:   os.Getenv(fmt.Sprintf("PROJECT_%d_CONTENT_TYPE", i)),
			CacheTTL:      time.Duration(ttl) * time.Second,
			IdPlaceholder: idPlaceholder,
			SourceType:    os.Getenv(fmt.Sprintf("PROJECT_%d_SOURCE_TYPE", i)),
		}

		if project.SourceType == "" {
			project.SourceType = "database" // Default source type
		}

		// Load source-specific config and validate
		switch project.SourceType {
		case "database":
			project.DB_DSN = os.Getenv(fmt.Sprintf("PROJECT_%d_DB_DSN", i))
			project.Table = os.Getenv(fmt.Sprintf("PROJECT_%d_TABLE", i))
			project.ServeColumn = os.Getenv(fmt.Sprintf("PROJECT_%d_SERVE_COLUMN", i))
			if project.DB_DSN == "" || project.Table == "" || project.ServeColumn == "" {
				return nil, fmt.Errorf("missing required database configuration (DB_DSN, TABLE, SERVE_COLUMN) for project %d", i)
			}
		case "api":
			project.APIEndpoint = os.Getenv(fmt.Sprintf("PROJECT_%d_API_ENDPOINT", i))
			project.APIAuthType = os.Getenv(fmt.Sprintf("PROJECT_%d_API_AUTH_TYPE", i))
			project.APIAuthSecret = os.Getenv(fmt.Sprintf("PROJECT_%d_API_AUTH_SECRET", i))
			project.APIAuthHeaderName = os.Getenv(fmt.Sprintf("PROJECT_%d_API_AUTH_HEADER_NAME", i))

			if project.APIEndpoint == "" {
				return nil, fmt.Errorf("missing required API configuration (API_ENDPOINT) for project %d", i)
			}
			if project.APIAuthType == "" {
				project.APIAuthType = "none"
			}

			// Validate auth config
			switch project.APIAuthType {
			case "bearer", "header":
				if project.APIAuthSecret == "" {
					return nil, fmt.Errorf("API_AUTH_SECRET must be set for auth type '%s' on project %d", project.APIAuthType, i)
				}
				if project.APIAuthType == "header" && project.APIAuthHeaderName == "" {
					return nil, fmt.Errorf("API_AUTH_HEADER_NAME must be set for auth type 'header' on project %d", i)
				}
			case "none":
				// No validation needed
			default:
				return nil, fmt.Errorf("unknown API_AUTH_TYPE '%s' for project %d", project.APIAuthType, i)
			}

		default:
			return nil, fmt.Errorf("unknown SOURCE_TYPE '%s' for project %d", project.SourceType, i)
		}

		// Basic validation
		if project.IdColumn == "" {
			return nil, fmt.Errorf("missing required configuration (ID_COLUMN) for project %d", i)
		}
		// Validate that the placeholder from the route matches the ID column
		if project.IdPlaceholder != project.IdColumn {
			return nil, fmt.Errorf("route placeholder {%s} must match ID_COLUMN '%s' for project %d", project.IdPlaceholder, project.IdColumn, i)
		}

		appConfig.Projects = append(appConfig.Projects, project)
	}

	if len(appConfig.Projects) == 0 {
		fmt.Println("Warning: No projects configured. The server will start with no active routes.")
	}

	return appConfig, nil
}

// Finds the placeholder in a route pattern.
// e.g., "/api/users/{user_id}/avatar" -> "user_id", nil
func extractIDPlaceholder(route string) (string, error) {
	start := strings.Index(route, "{")
	if start == -1 {
		return "", fmt.Errorf("no '{' found in route")
	}
	end := strings.Index(route, "}")
	if end == -1 || end < start {
		return "", fmt.Errorf("no '}' found after '{' in route")
	}
	return route[start+1 : end], nil
}
