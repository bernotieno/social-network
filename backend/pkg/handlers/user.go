package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bernaotieno/social-network/backend/pkg/middleware"
	"github.com/bernaotieno/social-network/backend/pkg/models"
	"github.com/bernaotieno/social-network/backend/pkg/utils"
	"github.com/gorilla/mux"
)

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	FullName  string `json:"fullName"`
	Bio       string `json:"bio"`
	IsPrivate bool   `json:"isPrivate"`
}

// GetUsers handles retrieving a list of users
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
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

	// Get current user ID from context (if authenticated)
	currentUserID, _ := middleware.GetUserID(r)

	// Get user
	user, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Remove password from response
	user.Password = ""

	// Get follower and following counts
	followersCount, err := h.FollowService.GetFollowersCount(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get followers count")
		return
	}

	followingCount, err := h.FollowService.GetFollowingCount(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get following count")
		return
	}

	// Check if current user is following this user
	isFollowing := false
	if currentUserID != "" && currentUserID != userID {
		isFollowing, err = h.FollowService.IsFollowing(currentUserID, userID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check follow status")
			return
		}
	}

	// Create response
	response := map[string]interface{}{
		"user":           user,
		"followersCount": followersCount,
		"followingCount": followingCount,
	}

	// Add follow status if authenticated
	if currentUserID != "" {
		response["isFollowedByCurrentUser"] = isFollowing
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

	// Update user fields
	user.FullName = req.FullName
	user.Bio = req.Bio
	user.IsPrivate = req.IsPrivate

	// Save changes
	if err := h.UserService.Update(user); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	// Remove password from response
	user.Password = ""

	utils.RespondWithSuccess(w, http.StatusOK, "Profile updated successfully", map[string]interface{}{
		"user": user,
	})
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

	if followingUser.IsPrivate {
		notificationType = models.NotificationTypeFollowRequest
		notificationContent = "requested to follow you"
	} else {
		notificationType = models.NotificationTypeNewFollower
		notificationContent = "started following you"
	}

	notification := &models.Notification{
		UserID:   followingID,
		SenderID: followerID,
		Type:     notificationType,
		Content:  notificationContent,
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

	// Get followers
	followers, err := h.FollowService.GetFollowers(userID, limit, offset)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get followers")
		return
	}

	// Remove passwords from response
	for _, user := range followers {
		user.Password = ""
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Followers retrieved successfully", map[string]interface{}{
		"followers": followers,
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

	// Get following
	following, err := h.FollowService.GetFollowing(userID, limit, offset)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get following")
		return
	}

	// Remove passwords from response
	for _, user := range following {
		user.Password = ""
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Following retrieved successfully", map[string]interface{}{
		"following": following,
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
