package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bernaotieno/social-network/backend/pkg/middleware"
	"github.com/bernaotieno/social-network/backend/pkg/models"
	"github.com/bernaotieno/social-network/backend/pkg/utils"
	"github.com/bernaotieno/social-network/backend/pkg/websocket"
	"github.com/gorilla/mux"
)

// AddCommentRequest represents a request to add a comment
type AddCommentRequest struct {
	Content string `json:"content"`
}

// GetComments handles retrieving comments for a post
func (h *Handler) GetComments(w http.ResponseWriter, r *http.Request) {
	// Get post ID from URL
	vars := mux.Vars(r)
	postID := vars["id"]

	// Get current user ID from context (if authenticated)
	currentUserID, _ := middleware.GetUserID(r)

	// Check if post exists and user can view it
	_, err := h.PostService.GetByID(postID, currentUserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Set default values
	limit := 50
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

	// Get comments
	comments, err := h.CommentService.GetCommentsByPost(postID, limit, offset)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get comments")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Comments retrieved successfully", map[string]interface{}{
		"comments": comments,
	})
}

// AddComment handles adding a comment to a post
func (h *Handler) AddComment(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get post ID from URL
	vars := mux.Vars(r)
	postID := vars["id"]

	// Parse request body
	var req AddCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate content
	if req.Content == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Content is required")
		return
	}

	// Check if post exists and user can view it
	post, err := h.PostService.GetByID(postID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Create comment
	comment := &models.Comment{
		PostID:  postID,
		UserID:  userID,
		Content: req.Content,
	}

	if err := h.CommentService.Create(comment); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create comment")
		return
	}

	// Get user for response
	user, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	user.Password = ""

	// Add user to comment for response
	comment.Author = user

	// Create notification for post owner (if not the same user)
	if post.UserID != userID {
		notification := &models.Notification{
			UserID:   post.UserID,
			SenderID: userID,
			Type:     "post_comment",
			Content:  "commented on your post",
			Data:     `{"postId":"` + postID + `","comment":"` + req.Content + `"}`,
		}

		if err := h.NotificationService.Create(notification); err != nil {
			// Log error but don't fail the request
			// TODO: Add proper logging
		}
	}

	// Broadcast new comment event via WebSocket
	newCommentEvent := map[string]interface{}{
		"postId":  postID,
		"comment": comment,
	}

	message := &websocket.Message{
		Type:    "new_comment",
		Content: newCommentEvent,
	}

	messageData, _ := json.Marshal(message)

	// Broadcast to all connected clients
	h.Hub.Broadcast <- &websocket.Broadcast{
		RoomID:  "", // Broadcast to default room (all users)
		Message: messageData,
		Sender:  nil, // No specific sender for server events
	}

	utils.RespondWithSuccess(w, http.StatusCreated, "Comment added successfully", map[string]interface{}{
		"comment": comment,
	})
}

// DeleteComment handles deleting a comment
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get post ID and comment ID from URL
	vars := mux.Vars(r)
	postID := vars["postId"]
	commentID := vars["commentId"]

	// Check if post exists and user can view it
	_, err = h.PostService.GetByID(postID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Delete comment
	if err := h.CommentService.Delete(commentID, userID); err != nil {
		if err.Error() == "comment not found or not authorized to delete" {
			utils.RespondWithError(w, http.StatusForbidden, "Not authorized to delete this comment")
		} else {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete comment")
		}
		return
	}

	// Broadcast comment deletion event via WebSocket
	deleteCommentEvent := map[string]interface{}{
		"postId":    postID,
		"commentId": commentID,
	}

	message := &websocket.Message{
		Type:    "comment_deleted",
		Content: deleteCommentEvent,
	}

	messageData, _ := json.Marshal(message)

	// Broadcast to all connected clients
	h.Hub.Broadcast <- &websocket.Broadcast{
		RoomID:  "", // Broadcast to default room (all users)
		Message: messageData,
		Sender:  nil, // No specific sender for server events
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Comment deleted successfully", nil)
}
