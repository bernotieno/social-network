package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bernaotieno/social-network/backend/pkg/middleware"
	"github.com/bernaotieno/social-network/backend/pkg/models"
	"github.com/bernaotieno/social-network/backend/pkg/utils"
	"github.com/gorilla/mux"
)

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	FullName    string `json:"fullName"`
	DateOfBirth string `json:"dateOfBirth"` // Format: YYYY-MM-DD
	Bio         string `json:"bio"`
	IsPrivate   bool   `json:"isPrivate"`
}

// GetUsers handles retrieving a list of users
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Get current user ID from context (authenticated user required)
	_, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	// Parse query parameters
	query := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Set default values
	limit := 20
	offset := 0

	// Parse limit and offset
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get users
	users, err := h.UserService.GetUsers(query, limit, offset)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get users")
		return
	}

	// Remove passwords from response
	for _, user := range users {
		user.Password = ""
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Users retrieved successfully", map[string]interface{}{
		"users": users,
	})
}

// GetUser handles retrieving a user by ID
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	userID := vars["id"]

	// Get current user ID from context (authenticated user required)
	currentUserID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get user
	user, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Remove password from response
	user.Password = ""

	// Check if current user is following this user and get follow status
	isFollowing := false
	hasPendingRequest := false
	followStatus := ""
	if currentUserID != userID {
		isFollowing, err = h.FollowService.IsFollowing(currentUserID, userID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check follow status")
			return
		}

		// Check for pending request
		hasPendingRequest, err = h.FollowService.HasPendingRequest(currentUserID, userID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check pending request")
			return
		}

		// Get follow status if there's a relationship
		if isFollowing || hasPendingRequest {
			status, err := h.FollowService.GetFollowStatus(currentUserID, userID)
			if err == nil {
				followStatus = string(status)
			}
		}
	}

	// Check if the viewer is authorized to see full profile data
	isOwnProfile := currentUserID == userID
	isAuthorized := false
	// Users can always see their own full profile
	if isOwnProfile {
		isAuthorized = true
	} else if user.IsPrivate {
		// For private profiles, only followers can see full data
		isAuthorized = isFollowing
	} else {
		// For public profiles, everyone can see full data
		isAuthorized = true
	}

	// Create filtered user data based on authorization
	var userData map[string]interface{}

	if isOwnProfile {
		// For own profile, always return ALL data including sensitive information
		userData = map[string]interface{}{
			"id":             user.ID,
			"username":       user.Username,
			"email":          user.Email,
			"fullName":       user.FullName,
			"firstName":      user.FirstName,
			"lastName":       user.LastName,
			"dateOfBirth":    user.DateOfBirth,
			"bio":            user.Bio,
			"profilePicture": user.ProfilePicture,
			"coverPhoto":     user.CoverPhoto,
			"isPrivate":      user.IsPrivate,
			"createdAt":      user.CreatedAt,
			"updatedAt":      user.UpdatedAt,
			"isOwnProfile":   true,
		}
	} else if isAuthorized {
		// Full profile data for authorized viewers (followers of private profiles, or anyone for public profiles)
		// But exclude sensitive information like email
		userData = map[string]interface{}{
			"id":             user.ID,
			"username":       user.Username,
			"fullName":       user.FullName,
			"firstName":      user.FirstName,
			"lastName":       user.LastName,
			"dateOfBirth":    user.DateOfBirth,
			"bio":            user.Bio,
			"profilePicture": user.ProfilePicture,
			"coverPhoto":     user.CoverPhoto,
			"isPrivate":      user.IsPrivate,
			"createdAt":      user.CreatedAt,
			"updatedAt":      user.UpdatedAt,
			"isOwnProfile":   false,
		}
	} else {
		// Limited profile data for unauthorized viewers of private profiles
		userData = map[string]interface{}{
			"id":             user.ID,
			"username":       user.Username,
			"fullName":       user.FullName,
			"profilePicture": user.ProfilePicture,
			"coverPhoto":     user.CoverPhoto,
			"isPrivate":      user.IsPrivate,
			"createdAt":      user.CreatedAt,
			"isOwnProfile":   false,
		}
	}

	// Get follower and following counts (for own profile or authorized viewers)
	var followersCount, followingCount int
	if isOwnProfile || isAuthorized {
		followersCount, err = h.FollowService.GetFollowersCount(userID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get followers count")
			return
		}

		followingCount, err = h.FollowService.GetFollowingCount(userID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get following count")
			return
		}
	}

	// Create response
	response := map[string]interface{}{
		"user": userData,
	}

	// Add stats for own profile or authorized viewers
	if isOwnProfile || isAuthorized {
		response["followersCount"] = followersCount
		response["followingCount"] = followingCount
	}

	// Add follow status if not viewing own profile
	if !isOwnProfile {
		response["isFollowedByCurrentUser"] = isFollowing
		response["hasPendingFollowRequest"] = hasPendingRequest
		response["followStatus"] = followStatus
	}

	utils.RespondWithSuccess(w, http.StatusOK, "User retrieved successfully", response)
}

// UpdateProfile handles updating a user's profile
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get user
	user, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Validate and update username if provided
	if req.Username != "" && req.Username != user.Username {
		// Trim whitespace and validate
		req.Username = strings.TrimSpace(req.Username)
		if len(req.Username) < 3 {
			utils.RespondWithError(w, http.StatusBadRequest, "Username must be at least 3 characters long")
			return
		}

		// Check if username is already taken
		existingUser, err := h.UserService.GetByUsername(req.Username)
		if err == nil && existingUser.ID != userID {
			utils.RespondWithError(w, http.StatusConflict, "Username is already taken")
			return
		}

		user.Username = req.Username
	}

	// Validate and update email if provided
	if req.Email != "" && req.Email != user.Email {
		// Trim whitespace and validate
		req.Email = strings.TrimSpace(req.Email)
		if !isValidEmail(req.Email) {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid email format")
			return
		}

		// Check if email is already taken
		existingUser, err := h.UserService.GetByEmail(req.Email)
		if err == nil && existingUser.ID != userID {
			utils.RespondWithError(w, http.StatusConflict, "Email is already taken")
			return
		}

		user.Email = req.Email
	}

	// Validate and update date of birth if provided
	if req.DateOfBirth != "" {
		// Parse date
		dateOfBirth, err := time.Parse("2006-01-02", req.DateOfBirth)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
			return
		}

		// Check if date is not in the future
		if dateOfBirth.After(time.Now()) {
			utils.RespondWithError(w, http.StatusBadRequest, "Date of birth cannot be in the future")
			return
		}

		// Check if user is at least 13 years old (basic age validation)
		minAge := time.Now().AddDate(-13, 0, 0)
		if dateOfBirth.After(minAge) {
			utils.RespondWithError(w, http.StatusBadRequest, "You must be at least 13 years old")
			return
		}

		user.DateOfBirth = &dateOfBirth
	}

	// Update other fields
	if req.FullName != "" {
		user.FullName = strings.TrimSpace(req.FullName)
	}
	user.Bio = req.Bio
	user.IsPrivate = req.IsPrivate

	// Save changes
	if err := h.UserService.Update(user); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.username") {
			utils.RespondWithError(w, http.StatusConflict, "Username is already taken")
		} else if strings.Contains(err.Error(), "UNIQUE constraint failed: users.email") {
			utils.RespondWithError(w, http.StatusConflict, "Email is already taken")
		} else {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update profile")
		}
		return
	}

	// Remove password from response
	user.Password = ""

	utils.RespondWithSuccess(w, http.StatusOK, "Profile updated successfully", map[string]interface{}{
		"user": user,
	})
}

// isValidEmail validates email format
func isValidEmail(email string) bool {
	// Basic email validation
	return strings.Contains(email, "@") && strings.Contains(email, ".") && len(email) > 5
}

// UploadAvatar handles uploading a user's profile picture
func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	// Get file from form
	file, header, err := r.FormFile("avatar")
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	// Save image
	imagePath, err := utils.SaveImage(file, header, "avatars")
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get user
	user, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Delete old avatar if exists
	if user.ProfilePicture != "" {
		utils.DeleteImage(user.ProfilePicture)
	}

	// Update user's profile picture
	user.ProfilePicture = imagePath
	if err := h.UserService.Update(user); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update profile picture")
		return
	}

	// Remove password from response
	user.Password = ""

	utils.RespondWithSuccess(w, http.StatusOK, "Profile picture updated successfully", map[string]interface{}{
		"user": user,
	})
}

// UploadCoverPhoto handles uploading a user's cover photo
func (h *Handler) UploadCoverPhoto(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	// Get file from form
	file, header, err := r.FormFile("coverPhoto")
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	// Save image
	imagePath, err := utils.SaveImage(file, header, "covers")
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get user
	user, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Delete old cover photo if exists
	if user.CoverPhoto != "" {
		utils.DeleteImage(user.CoverPhoto)
	}

	// Update user's cover photo
	user.CoverPhoto = imagePath
	if err := h.UserService.Update(user); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update cover photo")
		return
	}

	// Remove password from response
	user.Password = ""

	utils.RespondWithSuccess(w, http.StatusOK, "Cover photo updated successfully", map[string]interface{}{
		"user": user,
	})
}

// FollowUser handles following a user
func (h *Handler) FollowUser(w http.ResponseWriter, r *http.Request) {
	// Get current user ID from context
	followerID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get target user ID from URL
	vars := mux.Vars(r)
	followingID := vars["id"]

	// Check if user is trying to follow themselves
	if followerID == followingID {
		utils.RespondWithError(w, http.StatusBadRequest, "Cannot follow yourself")
		return
	}

	// Get target user
	followingUser, err := h.UserService.GetByID(followingID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Create follow relationship
	follow, err := h.FollowService.Create(followerID, followingID, followingUser.IsPrivate)
	if err != nil {
		if err.Error() == "follow relationship already exists" {
			utils.RespondWithError(w, http.StatusConflict, "Already following or request pending")
		} else {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to follow user")
		}
		return
	}

	// Create notification for the target user
	var notificationType models.NotificationType
	var notificationContent string
	var notificationData string

	if followingUser.IsPrivate {
		notificationType = models.NotificationTypeFollowRequest
		notificationContent = "requested to follow you"
		// Store the follow request ID in the notification data
		notificationData = `{"followRequestId":"` + follow.ID + `"}`
	} else {
		notificationType = models.NotificationTypeNewFollower
		notificationContent = "started following you"
	}

	notification := &models.Notification{
		UserID:   followingID,
		SenderID: followerID,
		Type:     notificationType,
		Content:  notificationContent,
		Data:     notificationData,
	}

	if err := h.NotificationService.Create(notification); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Follow request sent", map[string]interface{}{
		"follow": follow,
	})
}

// UnfollowUser handles unfollowing a user
func (h *Handler) UnfollowUser(w http.ResponseWriter, r *http.Request) {
	// Get current user ID from context
	followerID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get target user ID from URL
	vars := mux.Vars(r)
	followingID := vars["id"]

	// Delete follow relationship
	if err := h.FollowService.Delete(followerID, followingID); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to unfollow user")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Unfollowed successfully", nil)
}

// GetFollowers handles retrieving a user's followers
func (h *Handler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	userID := vars["id"]

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Set default values
	limit := 20
	offset := 0

	// Parse limit and offset
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get current user ID from context (authenticated user required)
	currentUserID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check if the target user has a private profile
	targetUser, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Check if current user is authorized to view followers
	isOwnProfile := currentUserID == userID
	if targetUser.IsPrivate && !isOwnProfile {
		// For private profiles, check if current user is a follower
		isFollowing, err := h.FollowService.IsFollowing(currentUserID, userID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check follow status")
			return
		}
		
		if !isFollowing {
			utils.RespondWithError(w, http.StatusForbidden, "Not authorized to view followers of this private account")
			return
		}
	}

	// Get followers
	followers, err := h.FollowService.GetFollowers(userID, limit, offset)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get followers")
		return
	}

	// Create response with follow status for each follower
	followersWithStatus := make([]map[string]interface{}, len(followers))
	for i, user := range followers {
		user.Password = ""

		userMap := map[string]interface{}{
			"id":             user.ID,
			"username":       user.Username,
			"email":          user.Email,
			"fullName":       user.FullName,
			"bio":            user.Bio,
			"profilePicture": user.ProfilePicture,
			"coverPhoto":     user.CoverPhoto,
			"isPrivate":      user.IsPrivate,
			"createdAt":      user.CreatedAt,
			"updatedAt":      user.UpdatedAt,
			"isFollowing":    false, // Default to false
		}

		// Check if current user is following this follower
		if currentUserID != user.ID {
			isFollowing, err := h.FollowService.IsFollowing(currentUserID, user.ID)
			if err == nil {
				userMap["isFollowing"] = isFollowing
			}
		}

		followersWithStatus[i] = userMap
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Followers retrieved successfully", map[string]interface{}{
		"followers": followersWithStatus,
	})
}

// GetFollowing handles retrieving users a user is following
func (h *Handler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	userID := vars["id"]

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Set default values
	limit := 20
	offset := 0

	// Parse limit and offset
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get current user ID from context (authenticated user required)
	currentUserID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check if the target user has a private profile
	targetUser, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Check if current user is authorized to view following list
	isOwnProfile := currentUserID == userID
	if targetUser.IsPrivate && !isOwnProfile {
		// For private profiles, check if current user is a follower
		isFollowing, err := h.FollowService.IsFollowing(currentUserID, userID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check follow status")
			return
		}
		
		if !isFollowing {
			utils.RespondWithError(w, http.StatusForbidden, "Not authorized to view following list of this private account")
			return
		}
	}

	// Get following
	following, err := h.FollowService.GetFollowing(userID, limit, offset)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get following")
		return
	}

	// Create response with follow status for each followed user
	followingWithStatus := make([]map[string]interface{}, len(following))
	for i, user := range following {
		user.Password = ""

		userMap := map[string]interface{}{
			"id":             user.ID,
			"username":       user.Username,
			"email":          user.Email,
			"fullName":       user.FullName,
			"bio":            user.Bio,
			"profilePicture": user.ProfilePicture,
			"coverPhoto":     user.CoverPhoto,
			"isPrivate":      user.IsPrivate,
			"createdAt":      user.CreatedAt,
			"updatedAt":      user.UpdatedAt,
			"isFollowing":    true, // Default to true since this is the following list
		}

		// Check if current user is following this user (should be true for following list)
		if currentUserID != user.ID {
			isFollowing, err := h.FollowService.IsFollowing(currentUserID, user.ID)
			if err == nil {
				userMap["isFollowing"] = isFollowing
			}
		}

		followingWithStatus[i] = userMap
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Following retrieved successfully", map[string]interface{}{
		"following": followingWithStatus,
	})
}

// GetFollowRequests handles retrieving pending follow requests
func (h *Handler) GetFollowRequests(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get follow requests
	requests, err := h.FollowService.GetFollowRequests(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get follow requests")
		return
	}

	// Get user details for each request
	var requestsWithUsers []map[string]interface{}
	for _, request := range requests {
		user, err := h.UserService.GetByID(request.FollowerID)
		if err != nil {
			continue
		}
		user.Password = ""

		requestsWithUsers = append(requestsWithUsers, map[string]interface{}{
			"id":        request.ID,
			"createdAt": request.CreatedAt,
			"user":      user,
		})
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Follow requests retrieved successfully", map[string]interface{}{
		"requests": requestsWithUsers,
	})
}

// RespondToFollowRequest handles accepting or rejecting a follow request
type FollowRequestResponse struct {
	Accept bool `json:"accept"`
}

// RespondToFollowRequest handles accepting or rejecting a follow request
func (h *Handler) RespondToFollowRequest(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get request ID from URL
	vars := mux.Vars(r)
	requestID := vars["id"]

	// Parse request body
	var req FollowRequestResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get follow request
	follow, err := h.FollowService.GetByID(requestID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Follow request not found")
		return
	}

	// Check if the request is for the current user
	if follow.FollowingID != userID {
		utils.RespondWithError(w, http.StatusForbidden, "Not authorized to respond to this request")
		return
	}

	// Check if the request is pending
	if follow.Status != models.FollowStatusPending {
		utils.RespondWithError(w, http.StatusBadRequest, "Follow request is not pending")
		return
	}

	// Update follow status
	status := models.FollowStatusRejected
	if req.Accept {
		status = models.FollowStatusAccepted
	}

	if err := h.FollowService.UpdateStatus(requestID, status); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update follow request")
		return
	}

	// Create notification for the follower
	notification := &models.Notification{
		UserID:   follow.FollowerID,
		SenderID: userID,
		Type:     models.NotificationTypeFollowAccepted,
		Content:  "accepted your follow request",
	}

	if err := h.NotificationService.Create(notification); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Follow request updated successfully", nil)
}
