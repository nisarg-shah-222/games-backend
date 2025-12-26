package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/games-app/backend/internal/database"
)

// PartnerHandler handles partner-related requests
type PartnerHandler struct {
	userRepo        *database.UserRepository
	partnershipRepo *database.PartnershipRepository
}

// NewPartnerHandler creates a new partner handler
func NewPartnerHandler() *PartnerHandler {
	return &PartnerHandler{
		userRepo:        database.NewUserRepository(database.DB),
		partnershipRepo: database.NewPartnershipRepository(database.DB),
	}
}

// SendPartnerRequestRequest represents the request body for sending a partner request
type SendPartnerRequestRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// SendPartnerRequestResponse represents the response for sending a partner request
type SendPartnerRequestResponse struct {
	Request *database.PartnerRequest `json:"request"`
	Message string                   `json:"message"`
}

// SendPartnerRequest handles sending a partner request
func (h *PartnerHandler) SendPartnerRequest(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	senderUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	var req SendPartnerRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Check if user already has a partner
	hasPartnership, err := h.partnershipRepo.UserHasPartnership(senderUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check partnership status"})
		return
	}
	if hasPartnership {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You already have a partner"})
		return
	}

	// Check if user is trying to send request to themselves
	sender, err := h.userRepo.FindByID(senderUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		return
	}
	if sender.Email == req.Email {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot send a request to yourself"})
		return
	}

	// Check if request already exists
	existingRequest, err := h.partnershipRepo.FindRequestBySenderAndEmail(senderUUID, req.Email)
	if err == nil && existingRequest.Status == "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request already sent to this email"})
		return
	}

	// Find recipient by email (if they exist)
	recipient, err := h.userRepo.FindByEmail(req.Email)
	var recipientID *uuid.UUID
	if err == nil {
		recipientID = &recipient.ID
	}

	// Create partner request
	request := &database.PartnerRequest{
		SenderID:       senderUUID,
		RecipientEmail: req.Email,
		RecipientID:    recipientID,
		Status:         "pending",
	}

	if err := h.partnershipRepo.CreateRequest(request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request: " + err.Error()})
		return
	}

	// Load relations
	request, err = h.partnershipRepo.FindRequestByID(request.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load request"})
		return
	}

	c.JSON(http.StatusOK, SendPartnerRequestResponse{
		Request: request,
		Message: "Partner request sent successfully",
	})
}

// GetSentRequestsResponse represents the response for getting sent requests
type GetSentRequestsResponse struct {
	Requests []database.PartnerRequest `json:"requests"`
}

// GetSentRequests handles getting all sent partner requests
func (h *PartnerHandler) GetSentRequests(c *gin.Context) {
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

	requests, err := h.partnershipRepo.FindPendingRequestsBySender(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get requests: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, GetSentRequestsResponse{
		Requests: requests,
	})
}

// GetReceivedRequestsResponse represents the response for getting received requests
type GetReceivedRequestsResponse struct {
	Requests []database.PartnerRequest `json:"requests"`
}

// GetReceivedRequests handles getting all received partner requests
func (h *PartnerHandler) GetReceivedRequests(c *gin.Context) {
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

	// Get user to get their email for querying by email
	user, err := h.userRepo.FindByID(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user: " + err.Error()})
		return
	}

	// Query by both ID and email to handle requests sent before user signed up
	requests, err := h.partnershipRepo.FindPendingRequestsByRecipient(userUUID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get requests: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, GetReceivedRequestsResponse{
		Requests: requests,
	})
}

// AcceptPartnerRequestResponse represents the response for accepting a partner request
type AcceptPartnerRequestResponse struct {
	Partnership *database.Partnership `json:"partnership"`
	Message     string                `json:"message"`
}

// AcceptPartnerRequest handles accepting a partner request
func (h *PartnerHandler) AcceptPartnerRequest(c *gin.Context) {
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

	// Get user to verify email
	user, err := h.userRepo.FindByID(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	requestIDStr := c.Param("id")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	// Find the request
	request, err := h.partnershipRepo.FindRequestByID(requestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	// Verify the request is for this user (by ID or email)
	isRecipient := (request.RecipientID != nil && *request.RecipientID == userUUID) ||
		request.RecipientEmail == user.Email
	if !isRecipient {
		c.JSON(http.StatusForbidden, gin.H{"error": "This request is not for you"})
		return
	}

	// Check if request is still pending
	if request.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request is no longer pending"})
		return
	}

	// Check if user already has a partner
	hasPartnership, err := h.partnershipRepo.UserHasPartnership(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check partnership status"})
		return
	}
	if hasPartnership {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You already have a partner"})
		return
	}

	// Check if sender already has a partner
	hasPartnership, err = h.partnershipRepo.UserHasPartnership(request.SenderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check sender partnership status"})
		return
	}
	if hasPartnership {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sender already has a partner"})
		return
	}

	// Create partnership (ensure consistent ordering: smaller UUID first)
	user1ID := request.SenderID
	user2ID := userUUID
	if userUUID.String() < request.SenderID.String() {
		user1ID = userUUID
		user2ID = request.SenderID
	}

	partnership := &database.Partnership{
		User1ID: user1ID,
		User2ID: user2ID,
	}

	if err := h.partnershipRepo.CreatePartnership(partnership); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create partnership: " + err.Error()})
		return
	}

	// Update request status and set recipient_id if it wasn't set before
	request.Status = "accepted"
	if request.RecipientID == nil {
		request.RecipientID = &userUUID
	}
	request.UpdatedAt = time.Now()
	if err := h.partnershipRepo.UpdateRequest(request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update request status"})
		return
	}

	// Cancel all other pending requests for both users
	if err := h.partnershipRepo.CancelPendingRequestsByUser(userUUID); err != nil {
		// Log error but don't fail the request
		_ = err
	}
	if err := h.partnershipRepo.CancelPendingRequestsByUser(request.SenderID); err != nil {
		// Log error but don't fail the request
		_ = err
	}

	// Load partnership with relations
	partnership, err = h.partnershipRepo.FindPartnershipByUser(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load partnership"})
		return
	}

	c.JSON(http.StatusOK, AcceptPartnerRequestResponse{
		Partnership: partnership,
		Message:     "Partner request accepted successfully",
	})
}

// RejectPartnerRequestResponse represents the response for rejecting a partner request
type RejectPartnerRequestResponse struct {
	Message string `json:"message"`
}

// RejectPartnerRequest handles rejecting a partner request
func (h *PartnerHandler) RejectPartnerRequest(c *gin.Context) {
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

	// Get user to verify email
	user, err := h.userRepo.FindByID(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	requestIDStr := c.Param("id")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	// Find the request
	request, err := h.partnershipRepo.FindRequestByID(requestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	// Verify the request is for this user (by ID or email)
	isRecipient := (request.RecipientID != nil && *request.RecipientID == userUUID) ||
		request.RecipientEmail == user.Email
	if !isRecipient {
		c.JSON(http.StatusForbidden, gin.H{"error": "This request is not for you"})
		return
	}

	// Update request status
	request.Status = "rejected"
	request.UpdatedAt = time.Now()
	if err := h.partnershipRepo.UpdateRequest(request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update request status"})
		return
	}

	c.JSON(http.StatusOK, RejectPartnerRequestResponse{
		Message: "Partner request rejected",
	})
}

// CancelPartnerRequestResponse represents the response for cancelling a partner request
type CancelPartnerRequestResponse struct {
	Message string `json:"message"`
}

// CancelPartnerRequest handles cancelling a sent partner request
func (h *PartnerHandler) CancelPartnerRequest(c *gin.Context) {
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

	// Find the request
	request, err := h.partnershipRepo.FindRequestByID(requestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	// Verify the request was sent by this user
	if request.SenderID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only cancel your own requests"})
		return
	}

	// Update request status
	request.Status = "cancelled"
	request.UpdatedAt = time.Now()
	if err := h.partnershipRepo.UpdateRequest(request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update request status"})
		return
	}

	c.JSON(http.StatusOK, CancelPartnerRequestResponse{
		Message: "Partner request cancelled",
	})
}

// GetCurrentPartnerResponse represents the response for getting current partner
type GetCurrentPartnerResponse struct {
	Partnership *database.Partnership `json:"partnership"`
}

// GetCurrentPartner handles getting the current partner
func (h *PartnerHandler) GetCurrentPartner(c *gin.Context) {
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

	partnership, err := h.partnershipRepo.FindPartnershipByUser(userUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No partner found"})
		return
	}

	c.JSON(http.StatusOK, GetCurrentPartnerResponse{
		Partnership: partnership,
	})
}

// DisconnectPartnerResponse represents the response for disconnecting from partner
type DisconnectPartnerResponse struct {
	Message string `json:"message"`
}

// DisconnectPartner handles disconnecting from a partner
func (h *PartnerHandler) DisconnectPartner(c *gin.Context) {
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

	// Find partnership
	partnership, err := h.partnershipRepo.FindPartnershipByUser(userUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No partnership found"})
		return
	}

	// Delete partnership
	if err := h.partnershipRepo.DeletePartnership(partnership.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, DisconnectPartnerResponse{
		Message: "Disconnected from partner successfully",
	})
}
