package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/PythonicVarun/Stratum/internal/api"
	"github.com/PythonicVarun/Stratum/internal/cache"
	"github.com/PythonicVarun/Stratum/internal/config"
	"github.com/PythonicVarun/Stratum/internal/database"
	"github.com/PythonicVarun/Stratum/pkg/utils"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	if len(cfg.Projects) == 0 {
		log.Println("Warning: No projects configured. Server will start but serve no routes.")
	}

	dbManager := database.NewConnectionManager()

	var redisCache cache.Cache
	if cfg.RedisURL != "" {
		var err error
		redisCache, err = cache.NewRedisCache(cfg.RedisURL)
		if err != nil {
			log.Printf("Warning: Could not connect to Redis. Caching will be disabled. Error: %v", err)
			redisCache = &cache.NoOpCache{}
		}
	} else {
		log.Println("Warning: REDIS_URL not set. Caching is disabled.")
		redisCache = &cache.NoOpCache{}
	}

	server := api.NewServer(cfg, dbManager, redisCache)

	go func() {
		server.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	utils.StratumLog("INFO", "Server is shutting down...")

	dbManager.CloseAll()
	if err := redisCache.Close(); err != nil {
		utils.StratumLog("INFO", "Error closing Redis cache: %v", err)
	}

	utils.StratumLog("INFO", "Server gracefully stopped.")
}
