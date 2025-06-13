package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/bernaotieno/social-network/backend/pkg/middleware"
	"github.com/bernaotieno/social-network/backend/pkg/models"
	"github.com/bernaotieno/social-network/backend/pkg/utils"
	"github.com/bernaotieno/social-network/backend/pkg/websocket"
	"github.com/gorilla/mux"
)

// CreatePostRequest represents a request to create a post
type CreatePostRequest struct {
	Content       string                `json:"content"`
	Visibility    models.PostVisibility `json:"visibility"`
	CustomViewers []string              `json:"customViewers,omitempty"`
}

// UpdatePostRequest represents a request to update a post
type UpdatePostRequest struct {
	Content    string                `json:"content"`
	Visibility models.PostVisibility `json:"visibility"`
}

// CreatePost handles creating a new post
func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
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

	// Get form values
	content := r.FormValue("content")
	visibility := r.FormValue("visibility")

	// Validate content
	if content == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Content is required")
		return
	}

	// Validate visibility
	if visibility != string(models.PostVisibilityPublic) &&
		visibility != string(models.PostVisibilityFollowers) &&
		visibility != string(models.PostVisibilityPrivate) &&
		visibility != string(models.PostVisibilityCustom) {
		visibility = string(models.PostVisibilityPublic)
	}

	// Create post
	post := &models.Post{
		UserID:     userID,
		Content:    content,
		Visibility: models.PostVisibility(visibility),
	}

	// Check if image was uploaded
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		// Save image
		imagePath, err := utils.SaveImage(file, header, "posts")
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		post.Image = imagePath
	}

	// Save post
	if err := h.PostService.Create(post); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create post")
		return
	}

	// Handle custom viewers if visibility is custom
	if post.Visibility == models.PostVisibilityCustom {
		customViewersStr := r.FormValue("customViewers")
		if customViewersStr != "" {
			// Parse custom viewers from JSON string
			var customViewers []string
			if err := json.Unmarshal([]byte(customViewersStr), &customViewers); err != nil {
				// If JSON parsing fails, try comma-separated values
				customViewers = []string{}
				for _, viewer := range strings.Split(customViewersStr, ",") {
					viewer = strings.TrimSpace(viewer)
					if viewer != "" {
						customViewers = append(customViewers, viewer)
					}
				}
			}

			// Add viewers to the post
			if len(customViewers) > 0 {
				if err := h.PostViewerService.AddViewers(post.ID, customViewers); err != nil {
					// Log error but don't fail the post creation
					// TODO: Add proper logging
				}
			}
		}
	}

	// Get user for response
	user, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	user.Password = ""

	// Add user to post for response
	post.User = user

	// Broadcast new post event via WebSocket (only for public posts)
	if post.Visibility == models.PostVisibilityPublic {
		newPostEvent := map[string]interface{}{
			"post": post,
		}

		message := &websocket.Message{
			Type:    "new_post",
			Content: newPostEvent,
		}

		messageData, _ := json.Marshal(message)

		// Broadcast to all connected clients
		h.Hub.Broadcast <- &websocket.Broadcast{
			RoomID:  "", // Broadcast to default room (all users)
			Message: messageData,
			Sender:  nil, // No specific sender for server events
		}
	}

	utils.RespondWithSuccess(w, http.StatusCreated, "Post created successfully", map[string]interface{}{
		"post": post,
	})
}

// GetPost handles retrieving a post by ID
func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	// Get post ID from URL
	vars := mux.Vars(r)
	postID := vars["id"]

	// Get current user ID from context (if authenticated)
	currentUserID, _ := middleware.GetUserID(r)

	// Get post
	post, err := h.PostService.GetByID(postID, currentUserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Post retrieved successfully", map[string]interface{}{
		"post": post,
	})
}

// UpdatePost handles updating a post
func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get post ID from URL
	vars := mux.Vars(r)
	postID := vars["id"]

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	// Get form values
	content := r.FormValue("content")
	visibility := r.FormValue("visibility")

	// Validate content
	if content == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Content is required")
		return
	}

	// Validate visibility
	if visibility != string(models.PostVisibilityPublic) &&
		visibility != string(models.PostVisibilityFollowers) &&
		visibility != string(models.PostVisibilityPrivate) &&
		visibility != string(models.PostVisibilityCustom) {
		visibility = string(models.PostVisibilityPublic)
	}

	// Get post
	post, err := h.PostService.GetByID(postID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Check if user is the post owner
	if post.UserID != userID {
		utils.RespondWithError(w, http.StatusForbidden, "Not authorized to update this post")
		return
	}

	// Update post fields
	post.Content = content
	post.Visibility = models.PostVisibility(visibility)

	// Save changes
	if err := h.PostService.Update(post); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update post")
		return
	}

	// Get user for response
	user, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	user.Password = ""

	// Add user to post for response
	post.User = user

	utils.RespondWithSuccess(w, http.StatusOK, "Post updated successfully", map[string]interface{}{
		"post": post,
	})
}

// DeletePost handles deleting a post
func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get post ID from URL
	vars := mux.Vars(r)
	postID := vars["id"]

	// Get post to check ownership and get image path
	post, err := h.PostService.GetByID(postID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Check if user is the post owner
	if post.UserID != userID {
		utils.RespondWithError(w, http.StatusForbidden, "Not authorized to delete this post")
		return
	}

	// Delete post
	if err := h.PostService.Delete(postID, userID); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete post")
		return
	}

	// Delete post image if exists
	if post.Image != "" {
		utils.DeleteImage(post.Image)
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Post deleted successfully", nil)
}

// GetUserPosts handles retrieving posts by a user
func (h *Handler) GetUserPosts(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	userID := vars["id"]

	// Get current user ID from context (if authenticated)
	currentUserID, _ := middleware.GetUserID(r)

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

	// Get posts
	posts, err := h.PostService.GetUserPosts(userID, currentUserID, limit, offset)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get posts")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Posts retrieved successfully", map[string]interface{}{
		"posts": posts,
	})
}

// GetFeed handles retrieving posts for a user's feed
func (h *Handler) GetFeed(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

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

	// Get feed
	posts, err := h.PostService.GetFeed(userID, limit, offset)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get feed")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Feed retrieved successfully", map[string]interface{}{
		"posts": posts,
	})
}

// LikePost handles liking a post
func (h *Handler) LikePost(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get post ID from URL
	vars := mux.Vars(r)
	postID := vars["id"]

	// Check if post exists and user can view it
	post, err := h.PostService.GetByID(postID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Like post
	if err := h.LikeService.Create(postID, userID); err != nil {
		if err.Error() == "post already liked by user" {
			utils.RespondWithError(w, http.StatusConflict, "Post already liked")
		} else {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to like post")
		}
		return
	}

	// Create notification for post owner (if not the same user)
	if post.UserID != userID {
		notification := &models.Notification{
			UserID:   post.UserID,
			SenderID: userID,
			Type:     "post_like",
			Content:  "liked your post",
			Data:     `{"postId":"` + postID + `"}`,
		}

		if err := h.NotificationService.Create(notification); err != nil {
			// Log error but don't fail the request
			// TODO: Add proper logging
		}
	}

	// Broadcast like event via WebSocket
	likeEvent := map[string]interface{}{
		"postId": postID,
		"userId": userID,
		"action": "like",
	}

	message := &websocket.Message{
		Type:    "post_like",
		Content: likeEvent,
	}

	messageData, _ := json.Marshal(message)

	// Broadcast to all connected clients
	h.Hub.Broadcast <- &websocket.Broadcast{
		RoomID:  "", // Broadcast to default room (all users)
		Message: messageData,
		Sender:  nil, // No specific sender for server events
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Post liked successfully", nil)
}

// UnlikePost handles unliking a post
func (h *Handler) UnlikePost(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get post ID from URL
	vars := mux.Vars(r)
	postID := vars["id"]

	// Unlike post
	if err := h.LikeService.Delete(postID, userID); err != nil {
		if err.Error() == "like not found" {
			utils.RespondWithError(w, http.StatusNotFound, "Post not liked")
		} else {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to unlike post")
		}
		return
	}

	// Broadcast unlike event via WebSocket
	unlikeEvent := map[string]interface{}{
		"postId": postID,
		"userId": userID,
		"action": "unlike",
	}

	message := &websocket.Message{
		Type:    "post_like",
		Content: unlikeEvent,
	}

	messageData, _ := json.Marshal(message)

	// Broadcast to all connected clients
	h.Hub.Broadcast <- &websocket.Broadcast{
		RoomID:  "", // Broadcast to default room (all users)
		Message: messageData,
		Sender:  nil, // No specific sender for server events
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Post unliked successfully", nil)
}
