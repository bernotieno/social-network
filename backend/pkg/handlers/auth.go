package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/bernaotieno/social-network/backend/pkg/auth"
	"github.com/bernaotieno/social-network/backend/pkg/middleware"
	"github.com/bernaotieno/social-network/backend/pkg/models"
	"github.com/bernaotieno/social-network/backend/pkg/utils"
)

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"fullName"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles user registration
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	log.Println("register request:", req)

	// Validate request
	if req.Username == "" || req.Email == "" || req.Password == "" || req.FullName == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "All fields are required")
		return
	}

	// Validate email format
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid email format")
		return
	}

	// Validate password length
	if len(req.Password) < 6 {
		utils.RespondWithError(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	// Check if user already exists
	exists, err := h.UserService.UserExists(req.Email, req.Username)
	log.Println("user exist", exists)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check user existence")
		return
	}
	if exists {
		utils.RespondWithError(w, http.StatusConflict, "User with this email or username already exists")
		return
	}

	// Create user
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	}
	if err := h.UserService.Create(user); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Create session
	sessionID, err := auth.CreateSession(r.Context(), h.DB, user.ID, w, r)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Return user data (without password)
	user.Password = ""
	utils.RespondWithSuccess(w, http.StatusCreated, "User registered successfully", map[string]interface{}{
		"user":  user,
		"token": sessionID,
	})
}

// Login handles user login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	log.Println("login request:", req)

	// Validate request
	if req.Email == "" || req.Password == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	// Get user by email
	user, err := h.UserService.GetByEmail(req.Email)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Check password
	if !h.UserService.CheckPassword(user, req.Password) {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Create session
	sessionID, err := auth.CreateSession(r.Context(), h.DB, user.ID, w, r)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Return user data (without password)
	user.Password = ""
	utils.RespondWithSuccess(w, http.StatusOK, "Login successful", map[string]interface{}{
		"user":  user,
		"token": sessionID,
	})
}

// Logout handles user logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Clear session
	if err := auth.ClearSession(r.Context(), h.DB, w, r); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to logout")
		return
	}

	// Delete all sessions for the user
	if err := h.SessionService.DeleteAllForUser(userID); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete sessions")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Logout successful", nil)
}
