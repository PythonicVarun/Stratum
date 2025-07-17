package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/PythonicVarun/Stratum/internal/cache"
	"github.com/PythonicVarun/Stratum/internal/config"
	"github.com/PythonicVarun/Stratum/internal/database"
	"github.com/PythonicVarun/Stratum/internal/datasource"
	"github.com/PythonicVarun/Stratum/pkg/utils"
	"github.com/gin-gonic/gin"
)

type Server struct {
	config    *config.AppConfig
	dbManager *database.ConnectionManager
	cache     cache.Cache
	router    *gin.Engine
}

// Creates and configures a new server instance.
func NewServer(cfg *config.AppConfig, dbManager *database.ConnectionManager, cache cache.Cache) *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	s := &Server{
		config:    cfg,
		dbManager: dbManager,
		cache:     cache,
		router:    router,
	}

	s.setupRoutes()
	return s
}

// Configures the router with all the dynamic project routes.
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Dynamically register routes from config
	for _, p := range s.config.Projects {
		project := p

		utils.StratumLog("INFO", "Registering route for project '%s': %s", project.Name, project.Route)

		// Convert placeholders {id} to gin-style :id
		ginRoute := convertToGinRoute(project.Route)
		s.router.GET(ginRoute, s.createHandler(project))
	}
}

// Returns a new gin.HandlerFunc for a given project configuration.
func (s *Server) createHandler(p config.Project) gin.HandlerFunc {
	source, err := datasource.NewDataSource(p, s.dbManager, s.config)
	if err != nil {
		utils.StratumLog("FATAL", "Could not create data source for project '%s': %v", p.Name, err)
		os.Exit(1)
	}

	return func(c *gin.Context) {
		idValue := strings.TrimPrefix(c.Param(p.IdPlaceholder), "/")

		endPlaceholder := strings.Index(p.Route, "}")
		if endPlaceholder != -1 && endPlaceholder < len(p.Route)-1 {
			suffix := p.Route[endPlaceholder+1:]
			idValue = strings.TrimSuffix(idValue, suffix)
		}

		if idValue == "" {
			c.String(http.StatusBadRequest, "ID not found in URL")
			return
		}

		ctx := c.Request.Context()
		cacheKey := fmt.Sprintf("%s:%s", p.Name, idValue)

		// Check for cache-bypassing headers
		pragmaHeader := c.GetHeader("Pragma")
		cacheControlHeader := c.GetHeader("Cache-Control")
		bypassCache := pragmaHeader == "no-cache" || strings.Contains(cacheControlHeader, "no-cache")

		if !bypassCache {
			cachedData, err := s.cache.Get(ctx, cacheKey)
			if err != nil {
				utils.StratumLog("ERROR", "Cache lookup failed for key '%s': %v", cacheKey, err)
			}

			if cachedData != nil {
				utils.StratumLog("INFO", "CACHE HIT: Serving '%s' from cache.", cacheKey)
				c.Header("Content-Type", p.ContentType)
				c.Header("X-Cache-Status", "HIT")
				c.Header("Cache-Control", fmt.Sprintf("public, max-age=%.0f", p.CacheTTL.Seconds()))
				c.Data(http.StatusOK, p.ContentType, cachedData)
				return
			}
		}

		if bypassCache {
			utils.StratumLog("INFO", "CACHE BYPASS: Client headers triggered cache bypass for key '%s'.", cacheKey)
			c.Header("X-Cache-Status", "BYPASS")
		} else {
			utils.StratumLog("INFO", "CACHE MISS: Key '%s' not found.", cacheKey)
			c.Header("X-Cache-Status", "MISS")
		}

		data, err := source.Fetch(idValue)
		if err != nil {
			utils.StratumLog("ERROR", "Data source fetch failed for project '%s': %v", p.Name, err)
			c.String(http.StatusInternalServerError, "Internal Server Error!")
			return
		}

		if data == nil {
			c.String(http.StatusNotFound, "Not Found")
			return
		}

		err = s.cache.Set(ctx, cacheKey, data, p.CacheTTL)
		if err != nil {
			utils.StratumLog("ERROR", "Failed to set cache for key '%s': %v", cacheKey, err)
		} else {
			utils.StratumLog("INFO", "CACHE SET: Stored key '%s' with TTL %s.", cacheKey, p.CacheTTL)
		}

		c.Header("Cache-Control", fmt.Sprintf("public, max-age=%.0f", p.CacheTTL.Seconds()))
		c.Data(http.StatusOK, p.ContentType, data)
	}
}

// Start runs the HTTP server.
func (s *Server) Start() {
	utils.StratumLog("INFO", "Server starting on port %s...", s.config.ServerPort)
	err := s.router.Run(":" + s.config.ServerPort)
	if err != nil {
		utils.StratumLog("FATAL", "Failed to start server: %v", err)
		os.Exit(1)
	}
}

// Converts a placeholders route (/path/{id}) to a gin-style route (/path/:id).
func convertToGinRoute(route string) string {
	start := strings.Index(route, "{")
	if start == -1 {
		return route
	}
	end := strings.Index(route, "}")
	if end == -1 {
		return route
	}

	placeholder := route[start+1 : end]
	if end < len(route)-1 {
		return route[:start] + "*" + placeholder
	}

	return route[:start] + ":" + placeholder
}
