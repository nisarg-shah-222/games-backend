package main

import (
	"log"
	"os"

	"github.com/games-app/backend/internal/config"
	"github.com/games-app/backend/internal/database"
	"github.com/games-app/backend/internal/handler"
	"github.com/games-app/backend/internal/router"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	if cfg.DatabaseURL != "" {
		if err := database.Init(cfg.DatabaseURL); err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
			os.Exit(1)
		}
		defer func() {
			if err := database.Close(); err != nil {
				log.Printf("Error closing database: %v", err)
			}
		}()
	} else {
		log.Println("Warning: DATABASE_URL not set, database features will be unavailable")
	}

	// Initialize router
	r := router.New()

	// Register handlers
	healthHandler := handler.NewHealthHandler()
	router.RegisterHealthRoutes(r, healthHandler)

	// Register auth handlers if database is available
	if cfg.DatabaseURL != "" {
		authHandler, err := handler.NewAuthHandler(cfg)
		if err != nil {
			log.Fatalf("Failed to initialize auth handler: %v", err)
			os.Exit(1)
		}
		router.RegisterAuthRoutes(r, authHandler)

		// Register partner handlers
		partnerHandler := handler.NewPartnerHandler()
		router.RegisterPartnerRoutes(r, partnerHandler, authHandler)

		// Register game handlers
		gamesHandler := handler.NewGamesHandler()
		router.RegisterGameRoutes(r, gamesHandler, authHandler)
	}

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
