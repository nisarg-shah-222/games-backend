package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/games-app/backend/internal/database"
)

// GamesHandler handles game-related requests
type GamesHandler struct {
	partnershipRepo *database.PartnershipRepository
	gameRepo        *database.GameRepository
	gameRequestRepo *database.GameRequestRepository
	playRepo        *database.PlayRepository
}

// NewGamesHandler creates a new games handler
func NewGamesHandler() *GamesHandler {
	return &GamesHandler{
		partnershipRepo: database.NewPartnershipRepository(database.DB),
		gameRepo:        database.NewGameRepository(database.DB),
		gameRequestRepo: database.NewGameRequestRepository(database.DB),
		playRepo:        database.NewPlayRepository(database.DB),
	}
}

// ListGamesResponse represents the response for listing games
type ListGamesResponse struct {
	Games []database.Game `json:"games"`
}

// ListGames handles listing all available games
func (h *GamesHandler) ListGames(c *gin.Context) {
	games, err := h.gameRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch games: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListGamesResponse{
		Games: games,
	})
}

// CreateGameRequestRequest represents the request body for creating a game request
type CreateGameRequestRequest struct {
	GameID string `json:"game_id" binding:"required"`
}

// CreateGameRequestResponse represents the response for creating a game request
type CreateGameRequestResponse struct {
	Request *database.GameRequest `json:"request"`
}

// PlayGameRequest represents the request body for playing a game
type PlayGameRequest struct {
	GameID string `json:"game_id" binding:"required"`
}

// PlayGameResponse represents the response for playing a game
type PlayGameResponse struct {
	Play    *database.Play        `json:"play,omitempty"`
	Request *database.GameRequest `json:"request,omitempty"`
}

// PlayGame handles starting or joining a game
// First checks if there's a live play, if not creates a game request
func (h *GamesHandler) PlayGame(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	var req PlayGameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	gameID, err := uuid.Parse(req.GameID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game ID"})
		return
	}

	// Verify game exists
	_, err = h.gameRepo.FindByID(gameID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	// Get user's partnership
	partnership, err := h.partnershipRepo.FindPartnershipByUser(userUUID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You don't have a partner"})
		return
	}

	// Determine partner ID
	var partnerID uuid.UUID
	if partnership.User1ID == userUUID {
		partnerID = partnership.User2ID
	} else {
		partnerID = partnership.User1ID
	}

	// First, check if there's already a live play for this game
	play, err := h.playRepo.FindLivePlayByPartners(partnership.User1ID, partnership.User2ID, gameID)
	if err == nil && play != nil {
		// There's a live play, return it
		c.JSON(http.StatusOK, PlayGameResponse{
			Play: play,
		})
		return
	}

	// No live play exists, check if there's already a pending request
	pendingRequests, err := h.gameRequestRepo.FindPendingRequestsByRequester(userUUID)
	if err == nil {
		for _, pr := range pendingRequests {
			if pr.GameID == gameID && pr.PartnerID == partnerID {
				c.JSON(http.StatusOK, PlayGameResponse{
					Request: &pr,
				})
				return
			}
		}
	}

	// Create game request (valid for 24 hours)
	request := &database.GameRequest{
		GameID:      gameID,
		RequesterID: userUUID,
		PartnerID:   partnerID,
		Status:      "pending",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	if err := h.gameRequestRepo.CreateRequest(request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request: " + err.Error()})
		return
	}

	// Load request with relations
	request, err = h.gameRequestRepo.FindRequestByID(request.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load request"})
		return
	}

	c.JSON(http.StatusOK, PlayGameResponse{
		Request: request,
	})
}

// CreateGameRequest handles creating a new game request
func (h *GamesHandler) CreateGameRequest(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	var req CreateGameRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	gameID, err := uuid.Parse(req.GameID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game ID"})
		return
	}

	// Verify game exists
	_, err = h.gameRepo.FindByID(gameID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	// Get user's partnership
	partnership, err := h.partnershipRepo.FindPartnershipByUser(userUUID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You don't have a partner"})
		return
	}

	// Determine partner ID
	var partnerID uuid.UUID
	if partnership.User1ID == userUUID {
		partnerID = partnership.User2ID
	} else {
		partnerID = partnership.User1ID
	}

	// Check if there's already a pending request
	pendingRequests, err := h.gameRequestRepo.FindPendingRequestsByRequester(userUUID)
	if err == nil {
		for _, pr := range pendingRequests {
			if pr.GameID == gameID && pr.PartnerID == partnerID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "You already have a pending request for this game"})
				return
			}
		}
	}

	// Create game request (valid for 24 hours)
	request := &database.GameRequest{
		GameID:      gameID,
		RequesterID: userUUID,
		PartnerID:   partnerID,
		Status:      "pending",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	if err := h.gameRequestRepo.CreateRequest(request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request: " + err.Error()})
		return
	}

	// Load request with relations
	request, err = h.gameRequestRepo.FindRequestByID(request.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load request"})
		return
	}

	c.JSON(http.StatusOK, CreateGameRequestResponse{
		Request: request,
	})
}

// GetPendingGameRequestsResponse represents the response for getting pending game requests
type GetPendingGameRequestsResponse struct {
	Requests []database.GameRequest `json:"requests"`
}

// GetPendingGameRequests handles getting pending game requests for the current user
func (h *GamesHandler) GetPendingGameRequests(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	// Expire old requests first
	_ = h.gameRequestRepo.ExpireOldRequests()

	// Get pending requests where user is the partner (received requests)
	requests, err := h.gameRequestRepo.FindPendingRequestsByPartner(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch requests: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, GetPendingGameRequestsResponse{
		Requests: requests,
	})
}

// RespondToGameRequestRequest represents the request body for responding to a game request
type RespondToGameRequestRequest struct {
	Accept bool `json:"accept"`
}

// RespondToGameRequestResponse represents the response for responding to a game request
type RespondToGameRequestResponse struct {
	Request *database.GameRequest `json:"request"`
	Play    *database.Play        `json:"play,omitempty"`
}

// RespondToGameRequest handles accepting or rejecting a game request
func (h *GamesHandler) RespondToGameRequest(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	requestIDStr := c.Param("id")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var req RespondToGameRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Get request
	request, err := h.gameRequestRepo.FindRequestByID(requestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	// Verify user is the partner
	if request.PartnerID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not the recipient of this request"})
		return
	}

	// Check if request is expired
	if request.IsExpired() {
		request.Status = "expired"
		h.gameRequestRepo.UpdateRequest(request)
		c.JSON(http.StatusBadRequest, gin.H{"error": "This request has expired"})
		return
	}

	// Check if already responded
	if request.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request has already been responded to"})
		return
	}

	if req.Accept {
		// Accept the request
		request.Status = "accepted"
		if err := h.gameRequestRepo.UpdateRequest(request); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update request: " + err.Error()})
			return
		}

		// End any existing live plays for this partner combination
		_ = h.playRepo.EndAllLivePlaysByPartners(request.RequesterID, request.PartnerID)

		// Create a new play
		play := &database.Play{
			GameID:     request.GameID,
			Partner1ID: request.RequesterID,
			Partner2ID: request.PartnerID,
			PlayData:   database.JSONB{},
			IsLive:     true,
		}

		if err := h.playRepo.CreatePlay(play); err != nil {
			// Rollback request status
			request.Status = "pending"
			h.gameRequestRepo.UpdateRequest(request)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create play: " + err.Error()})
			return
		}

		// Load play with relations
		play, err = h.playRepo.FindPlayByID(play.ID)
		if err != nil {
			// Play created but failed to load, still return success
			play = nil
		}

		c.JSON(http.StatusOK, RespondToGameRequestResponse{
			Request: request,
			Play:    play,
		})
	} else {
		// Reject the request
		request.Status = "rejected"
		if err := h.gameRequestRepo.UpdateRequest(request); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update request: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, RespondToGameRequestResponse{
			Request: request,
		})
	}
}

// GetLivePlayResponse represents the response for getting a live play
type GetLivePlayResponse struct {
	Play *database.Play `json:"play"`
}

// GetLivePlay handles getting the live play for a game and partnership
func (h *GamesHandler) GetLivePlay(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	gameIDStr := c.Param("gameId")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game ID"})
		return
	}

	// Get user's partnership
	partnership, err := h.partnershipRepo.FindPartnershipByUser(userUUID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You don't have a partner"})
		return
	}

	// Find live play
	play, err := h.playRepo.FindLivePlayByPartners(partnership.User1ID, partnership.User2ID, gameID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No live play found"})
		return
	}

	c.JSON(http.StatusOK, GetLivePlayResponse{
		Play: play,
	})
}

// UpdatePlayRequest represents the request body for updating a play
type UpdatePlayRequest struct {
	PlayData database.JSONB `json:"play_data" binding:"required"`
}

// UpdatePlayResponse represents the response for updating a play
type UpdatePlayResponse struct {
	Play *database.Play `json:"play"`
}

// UpdatePlay handles updating a play's data
func (h *GamesHandler) UpdatePlay(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	playIDStr := c.Param("id")
	playID, err := uuid.Parse(playIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid play ID"})
		return
	}

	var req UpdatePlayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Get play
	play, err := h.playRepo.FindPlayByID(playID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Play not found"})
		return
	}

	// Verify user is part of this play
	if play.Partner1ID != userUUID && play.Partner2ID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not part of this play"})
		return
	}

	// Update play data
	play.PlayData = req.PlayData
	if err := h.playRepo.UpdatePlay(play); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update play: " + err.Error()})
		return
	}

	// Reload play
	play, err = h.playRepo.FindPlayByID(playID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload play"})
		return
	}

	c.JSON(http.StatusOK, UpdatePlayResponse{
		Play: play,
	})
}

// GetPlayByIdResponse represents the response for getting a play by ID
type GetPlayByIdResponse struct {
	Play *database.Play `json:"play"`
}

// GetPlayById handles getting a play by ID
func (h *GamesHandler) GetPlayById(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	playIDStr := c.Param("id")
	playID, err := uuid.Parse(playIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid play ID"})
		return
	}

	// Get play
	play, err := h.playRepo.FindPlayByID(playID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Play not found"})
		return
	}

	// Verify user is part of this play
	if play.Partner1ID != userUUID && play.Partner2ID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not part of this play"})
		return
	}

	// For Bulls and Cows, hide opponent's secret until game is completed
	if play.GameID.String() == "550e8400-e29b-41d4-a716-446655440001" {
		playData := play.PlayData
		if playData != nil {
			// Determine which partner the user is
			isPartner1 := play.Partner1ID == userUUID

			// Hide opponent's secret if game is not completed
			if status, exists := playData["status"]; exists && status != "completed" {
				if isPartner1 {
					// Hide partner2's secret
					playData["partner2_secret"] = nil
				} else {
					// Hide partner1's secret
					playData["partner1_secret"] = nil
				}
				play.PlayData = playData
			}
		}
	}

	c.JSON(http.StatusOK, GetPlayByIdResponse{
		Play: play,
	})
}

// SetSecretRequest represents the request body for setting a secret
type SetSecretRequest struct {
	Secret string `json:"secret" binding:"required,len=4"`
}

// SetSecretResponse represents the response for setting a secret
type SetSecretResponse struct {
	Play *database.Play `json:"play"`
}

// validateSecret validates a 4-digit secret number
func validateSecret(secret string) error {
	if len(secret) != 4 {
		return fmt.Errorf("secret must be exactly 4 digits")
	}

	// Check for leading zero
	if secret[0] == '0' {
		return fmt.Errorf("secret cannot start with 0")
	}

	// Check all characters are digits
	for _, char := range secret {
		if char < '0' || char > '9' {
			return fmt.Errorf("secret must contain only digits")
		}
	}

	// Check for unique digits
	digits := make(map[rune]bool)
	for _, char := range secret {
		if digits[char] {
			return fmt.Errorf("secret must have unique digits")
		}
		digits[char] = true
	}

	return nil
}

// SetSecret handles setting a player's secret number
func (h *GamesHandler) SetSecret(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	playIDStr := c.Param("id")
	playID, err := uuid.Parse(playIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid play ID"})
		return
	}

	var req SetSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate secret
	if err := validateSecret(req.Secret); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get play
	play, err := h.playRepo.FindPlayByID(playID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Play not found"})
		return
	}

	// Verify user is part of this play
	if play.Partner1ID != userUUID && play.Partner2ID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not part of this play"})
		return
	}

	// Get play data
	playData := play.PlayData
	if playData == nil {
		playData = make(database.JSONB)
	}

	// Determine which partner the user is
	var secretKey string
	if play.Partner1ID == userUUID {
		secretKey = "partner1_secret"
	} else {
		secretKey = "partner2_secret"
	}

	// Check if secret already set
	if existingSecret, exists := playData[secretKey]; exists && existingSecret != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have already set your secret"})
		return
	}

	// Set the secret
	playData[secretKey] = req.Secret

	// Initialize status if not set
	if _, exists := playData["status"]; !exists {
		playData["status"] = "waiting_secrets"
	}

	// Check if both secrets are set
	partner1Secret, hasPartner1 := playData["partner1_secret"]
	partner2Secret, hasPartner2 := playData["partner2_secret"]

	if hasPartner1 && partner1Secret != nil && hasPartner2 && partner2Secret != nil {
		// Both secrets set, start the game
		playData["status"] = "playing"
		// Set initial turn to partner1
		if _, exists := playData["current_turn"]; !exists {
			playData["current_turn"] = play.Partner1ID.String()
		}
		// Initialize guesses array if not exists
		if _, exists := playData["guesses"]; !exists {
			playData["guesses"] = []interface{}{}
		}
	}

	// Update play
	play.PlayData = playData
	if err := h.playRepo.UpdatePlay(play); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update play: " + err.Error()})
		return
	}

	// Reload play
	play, err = h.playRepo.FindPlayByID(playID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload play"})
		return
	}

	c.JSON(http.StatusOK, SetSecretResponse{
		Play: play,
	})
}

// calculateBullsAndCows calculates bulls and cows for a guess
func calculateBullsAndCows(secret, guess string) (int, int) {
	bulls := 0
	cows := 0

	secretDigits := []rune(secret)
	guessDigits := []rune(guess)

	// Count bulls (correct digit in correct position)
	for i := 0; i < 4; i++ {
		if secretDigits[i] == guessDigits[i] {
			bulls++
		}
	}

	// Count cows (correct digit in wrong position)
	secretCount := make(map[rune]int)
	guessCount := make(map[rune]int)

	for i := 0; i < 4; i++ {
		if secretDigits[i] != guessDigits[i] {
			secretCount[secretDigits[i]]++
			guessCount[guessDigits[i]]++
		}
	}

	// Count matching digits (excluding bulls)
	for digit, count := range guessCount {
		if secretCount[digit] > 0 {
			cows += min(count, secretCount[digit])
		}
	}

	return bulls, cows
}

// MakeGuessRequest represents the request body for making a guess
type MakeGuessRequest struct {
	Guess string `json:"guess" binding:"required,len=4"`
}

// MakeGuessResponse represents the response for making a guess
type MakeGuessResponse struct {
	Play  *database.Play `json:"play"`
	Bulls int            `json:"bulls"`
	Cows  int            `json:"cows"`
}

// MakeGuess handles making a guess in Bulls and Cows
func (h *GamesHandler) MakeGuess(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	playIDStr := c.Param("id")
	playID, err := uuid.Parse(playIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid play ID"})
		return
	}

	var req MakeGuessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate guess
	if err := validateSecret(req.Guess); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get play
	play, err := h.playRepo.FindPlayByID(playID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Play not found"})
		return
	}

	// Verify user is part of this play
	if play.Partner1ID != userUUID && play.Partner2ID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not part of this play"})
		return
	}

	// Get play data
	playData := play.PlayData
	if playData == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid play data"})
		return
	}

	// Check game status
	status, exists := playData["status"]
	if !exists || status != "playing" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Game is not in playing state"})
		return
	}

	// Check if it's user's turn
	currentTurn, exists := playData["current_turn"]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game state"})
		return
	}

	currentTurnStr, ok := currentTurn.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game state"})
		return
	}

	if currentTurnStr != userUUID.String() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "It's not your turn"})
		return
	}

	// Determine which partner the user is and get opponent's secret
	var isPartner1 bool
	var opponentSecret string
	if play.Partner1ID == userUUID {
		isPartner1 = true
		opponentSecretRaw, exists := playData["partner2_secret"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Opponent has not set their secret yet"})
			return
		}
		opponentSecret, ok = opponentSecretRaw.(string)
		if !ok || opponentSecret == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Opponent has not set their secret yet"})
			return
		}
	} else {
		opponentSecretRaw, exists := playData["partner1_secret"]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Opponent has not set their secret yet"})
			return
		}
		opponentSecret, ok = opponentSecretRaw.(string)
		if !ok || opponentSecret == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Opponent has not set their secret yet"})
			return
		}
	}

	// Calculate bulls and cows
	bulls, cows := calculateBullsAndCows(opponentSecret, req.Guess)

	// Get guesses array
	guesses, exists := playData["guesses"]
	if !exists {
		guesses = []interface{}{}
	}
	guessesArray, ok := guesses.([]interface{})
	if !ok {
		guessesArray = []interface{}{}
	}

	// Add new guess
	newGuess := map[string]interface{}{
		"player_id": userUUID.String(),
		"guess":     req.Guess,
		"bulls":     bulls,
		"cows":      cows,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	guessesArray = append(guessesArray, newGuess)
	playData["guesses"] = guessesArray

	// Check if game is won (4 bulls)
	if bulls == 4 {
		playData["status"] = "completed"
		playData["winner_id"] = userUUID.String()
		play.IsLive = false
	} else {
		// Switch turn
		if isPartner1 {
			playData["current_turn"] = play.Partner2ID.String()
		} else {
			playData["current_turn"] = play.Partner1ID.String()
		}
	}

	// Update play
	play.PlayData = playData
	if err := h.playRepo.UpdatePlay(play); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update play: " + err.Error()})
		return
	}

	// Reload play
	play, err = h.playRepo.FindPlayByID(playID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload play"})
		return
	}

	c.JSON(http.StatusOK, MakeGuessResponse{
		Play:  play,
		Bulls: bulls,
		Cows:  cows,
	})
}
