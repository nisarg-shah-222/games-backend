package router

import (
	"github.com/games-app/backend/internal/handler"
	"github.com/games-app/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// New creates a new Gin router with middleware
func New() *gin.Engine {
	// Set Gin mode based on environment
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// Apply global middleware
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	return r
}

// RegisterHealthRoutes registers health check routes
func RegisterHealthRoutes(r *gin.Engine, healthHandler *handler.HealthHandler) {
	v1 := r.Group("/api/v1")
	{
		v1.GET("/health-check", healthHandler.HealthCheck)
	}
}

// RegisterAuthRoutes registers authentication routes
func RegisterAuthRoutes(r *gin.Engine, authHandler *handler.AuthHandler) {
	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			// Public routes
			auth.POST("/request-otp", authHandler.RequestOtp)
			auth.POST("/verify-otp", authHandler.VerifyOtp)

			// Protected routes
			protected := auth.Group("")
			protected.Use(middleware.AuthMiddleware(authHandler))
			{
				protected.GET("/me", authHandler.GetCurrentUser)
			}
		}
		// User profile routes
		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(authHandler))
		{
			users.PUT("/me", authHandler.UpdateProfile)
		}
	}
}

// RegisterPartnerRoutes registers partner-related routes
func RegisterPartnerRoutes(r *gin.Engine, partnerHandler *handler.PartnerHandler, authHandler *handler.AuthHandler) {
	v1 := r.Group("/api/v1")
	{
		partners := v1.Group("/partners")
		partners.Use(middleware.AuthMiddleware(authHandler))
		{
			// Partner requests
			partners.POST("/request", partnerHandler.SendPartnerRequest)
			partners.GET("/requests/sent", partnerHandler.GetSentRequests)
			partners.GET("/requests/received", partnerHandler.GetReceivedRequests)
			partners.POST("/accept/:id", partnerHandler.AcceptPartnerRequest)
			partners.POST("/reject/:id", partnerHandler.RejectPartnerRequest)
			partners.DELETE("/request/:id", partnerHandler.CancelPartnerRequest)

			// Current partner
			partners.GET("/current", partnerHandler.GetCurrentPartner)
			partners.DELETE("/current", partnerHandler.DisconnectPartner)
		}
	}
}

// RegisterGameRoutes registers game-related routes
func RegisterGameRoutes(r *gin.Engine, gamesHandler *handler.GamesHandler, authHandler *handler.AuthHandler) {
	v1 := r.Group("/api/v1")
	{
		games := v1.Group("/games")
		{
			// Public routes
			games.GET("", gamesHandler.ListGames)

			// Protected routes
			protected := games.Group("")
			protected.Use(middleware.AuthMiddleware(authHandler))
			{
				// Play game (checks for live play first, then creates request)
				protected.POST("/play", gamesHandler.PlayGame)
				// Game requests
				protected.POST("/requests", gamesHandler.CreateGameRequest)
				protected.GET("/requests/pending", gamesHandler.GetPendingGameRequests)
				protected.POST("/requests/:id/respond", gamesHandler.RespondToGameRequest)

				// Plays
				protected.GET("/:gameId/play", gamesHandler.GetLivePlay)
				protected.GET("/plays/:id", gamesHandler.GetPlayById)
				protected.PUT("/plays/:id", gamesHandler.UpdatePlay)
				protected.POST("/plays/:id/set-secret", gamesHandler.SetSecret)
				protected.POST("/plays/:id/guess", gamesHandler.MakeGuess)
			}
		}
	}
}
