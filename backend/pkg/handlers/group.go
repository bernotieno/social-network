package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/bernaotieno/social-network/backend/pkg/middleware"
	"github.com/bernaotieno/social-network/backend/pkg/models"
	"github.com/bernaotieno/social-network/backend/pkg/utils"
	"github.com/bernaotieno/social-network/backend/pkg/websocket"
	"github.com/gorilla/mux"
)

// CreateGroupRequest represents a request to create a group
type CreateGroupRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Privacy     models.GroupPrivacy `json:"privacy"`
}

// UpdateGroupRequest represents a request to update a group
type UpdateGroupRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Privacy     models.GroupPrivacy `json:"privacy"`
}

// CreateGroupPostRequest represents a request to create a group post
type CreateGroupPostRequest struct {
	Content string `json:"content"`
}

// CreateGroupEventRequest represents a request to create a group event
type CreateGroupEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Location    string `json:"location"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
}

// EventResponseRequest represents a request to respond to an event
type EventResponseRequest struct {
	Response models.EventResponseType `json:"response"`
}

// GetGroups handles retrieving a list of groups
func (h *Handler) GetGroups(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
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

	// Get groups
	var groups []*models.Group
	if query != "" {
		groups, err = h.GroupService.SearchGroups(query, userID, limit, offset)
	} else {
		groups, err = h.GroupService.GetGroups(userID, limit, offset)
	}

	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get groups")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Groups retrieved successfully", map[string]interface{}{
		"groups": groups,
	})
}

// CreateGroup handles creating a new group
func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
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
	name := r.FormValue("name")
	description := r.FormValue("description")
	privacy := r.FormValue("privacy")

	// Validate name
	if name == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Name is required")
		return
	}

	// Validate privacy
	if privacy != string(models.GroupPrivacyPublic) && privacy != string(models.GroupPrivacyPrivate) {
		privacy = string(models.GroupPrivacyPublic)
	}

	// Create group
	group := &models.Group{
		Name:        name,
		Description: description,
		CreatorID:   userID,
		Privacy:     models.GroupPrivacy(privacy),
	}

	// Check if cover photo was uploaded
	file, header, err := r.FormFile("coverPhoto")
	if err == nil {
		defer file.Close()

		// Save image
		imagePath, err := utils.SaveImage(file, header, "groups")
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		group.CoverPhoto = imagePath
	}

	// Save group
	if err := h.GroupService.Create(group); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create group")
		return
	}

	// Get user for response
	user, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	user.Password = ""

	// Add creator to group for response
	group.Creator = user
	group.MembersCount = 1
	group.IsJoined = true

	utils.RespondWithSuccess(w, http.StatusCreated, "Group created successfully", map[string]interface{}{
		"group": group,
	})
}

// GetGroup handles retrieving a group by ID
func (h *Handler) GetGroup(w http.ResponseWriter, r *http.Request) {
	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get current user ID from context
	currentUserID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group
	group, err := h.GroupService.GetByID(groupID, currentUserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Group not found")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Group retrieved successfully", map[string]interface{}{
		"group": group,
	})
}

// UpdateGroup handles updating a group
func (h *Handler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	// Get form values
	name := r.FormValue("name")
	description := r.FormValue("description")
	privacy := r.FormValue("privacy")

	// Validate name
	if name == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Name is required")
		return
	}

	// Validate privacy
	if privacy != string(models.GroupPrivacyPublic) && privacy != string(models.GroupPrivacyPrivate) {
		privacy = string(models.GroupPrivacyPublic)
	}

	// Get group
	group, err := h.GroupService.GetByID(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Group not found")
		return
	}

	// Check if user is the group creator
	if group.CreatorID != userID {
		utils.RespondWithError(w, http.StatusForbidden, "Not authorized to update this group")
		return
	}

	// Update group fields
	group.Name = name
	group.Description = description
	group.Privacy = models.GroupPrivacy(privacy)

	// Check if cover photo was uploaded
	file, header, err := r.FormFile("coverPhoto")
	if err == nil {
		defer file.Close()

		// Save image
		imagePath, err := utils.SaveImage(file, header, "groups")
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Delete old cover photo if exists
		if group.CoverPhoto != "" {
			utils.DeleteImage(group.CoverPhoto)
		}

		group.CoverPhoto = imagePath
	}

	// Save changes
	if err := h.GroupService.Update(group); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update group")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Group updated successfully", map[string]interface{}{
		"group": group,
	})
}

// PromoteGroupMember handles promoting a group member to admin
func (h *Handler) PromoteGroupMember(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (the caller)
	callerID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID and member ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]
	memberID := vars["memberId"]

	// Promote member to admin
	if err := h.GroupMemberService.PromoteToAdmin(groupID, memberID, callerID); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Group member promoted to admin successfully", nil)
}

// DemoteGroupMember handles demoting a group admin to a regular member
func (h *Handler) DemoteGroupMember(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (the caller)
	callerID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID and member ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]
	memberID := vars["memberId"]

	// Demote admin to member
	if err := h.GroupMemberService.DemoteFromAdmin(groupID, memberID, callerID); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Group member demoted successfully", nil)
}

// RemoveGroupMember handles removing a member from a group
func (h *Handler) RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (the caller)
	callerID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID and member ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]
	memberID := vars["memberId"]

	// Remove member
	if err := h.GroupMemberService.RemoveMember(groupID, memberID, callerID); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Group member removed successfully", nil)
}

// DeleteGroup handles deleting a group
func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Delete group
	if err := h.GroupService.Delete(groupID, userID); err != nil {
		if err.Error() == "not authorized to delete this group" {
			utils.RespondWithError(w, http.StatusForbidden, "Not authorized to delete this group")
		} else {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete group")
		}
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Group deleted successfully", nil)
}

// JoinGroup handles joining a group
func (h *Handler) JoinGroup(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get group
	group, err := h.GroupService.GetByID(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Group not found")
		return
	}

	// Check if user is already a member or has a pending request
	existingMember, err := h.GroupMemberService.GetByGroupAndUser(groupID, userID)
	if err == nil {
		// User already has a relationship with this group
		switch existingMember.Status {
		case models.GroupMemberStatusAccepted:
			utils.RespondWithError(w, http.StatusConflict, "Already a member of this group")
			return
		case models.GroupMemberStatusPending:
			utils.RespondWithError(w, http.StatusConflict, "Join request already pending")
			return
		case models.GroupMemberStatusRejected:
			// Allow user to request again by updating the existing record
			existingMember.Status = models.GroupMemberStatusPending
			existingMember.UpdatedAt = time.Now()
			if err := h.GroupMemberService.Update(existingMember); err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update join request")
				return
			}

			// Create notification for group admins
			h.createJoinRequestNotification(group, userID)

			utils.RespondWithSuccess(w, http.StatusOK, "Join request sent", map[string]interface{}{
				"status": models.GroupMemberStatusPending,
			})
			return
		}
	}

	// All groups now require admin approval for join requests
	status := models.GroupMemberStatusPending

	// Create group member with pending status
	member := &models.GroupMember{
		GroupID: groupID,
		UserID:  userID,
		Role:    models.GroupMemberRoleMember,
		Status:  status,
	}

	if err := h.GroupMemberService.Create(member); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create join request")
		return
	}

	// Create notification for group admins
	h.createJoinRequestNotification(group, userID)

	utils.RespondWithSuccess(w, http.StatusOK, "Join request sent", map[string]interface{}{
		"status": status,
	})
}

// createJoinRequestNotification creates a notification for group admins about a join request
func (h *Handler) createJoinRequestNotification(group *models.Group, requesterID string) {
	// Get all group admins
	admins, err := h.GroupMemberService.GetGroupAdmins(group.ID)
	if err != nil {
		return // Log error but don't fail the request
	}

	// Create notification for each admin
	for _, admin := range admins {
		notification := &models.Notification{
			UserID:   admin.UserID,
			SenderID: requesterID,
			Type:     models.NotificationTypeGroupJoinRequest,
			Content:  "requested to join your group",
			Data:     `{"groupId":"` + group.ID + `","groupName":"` + group.Name + `"}`,
		}

		if err := h.NotificationService.Create(notification); err != nil {
			// Log error but don't fail the request
			// TODO: Add proper logging
		}
	}
}

// LeaveGroup handles leaving a group
func (h *Handler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get group
	group, err := h.GroupService.GetByID(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Group not found")
		return
	}

	// Check if user is a member
	if !group.IsJoined {
		utils.RespondWithError(w, http.StatusBadRequest, "Not a member of this group")
		return
	}

	// Check if user is the creator
	if group.CreatorID == userID {
		utils.RespondWithError(w, http.StatusBadRequest, "Group creator cannot leave the group. Delete the group instead.")
		return
	}

	// Leave group
	if err := h.GroupMemberService.Delete(groupID, userID); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to leave group")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Left group successfully", nil)
}

// GetGroupPosts handles retrieving posts for a group
func (h *Handler) GetGroupPosts(w http.ResponseWriter, r *http.Request) {
	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get current user ID from context
	currentUserID, err := middleware.GetUserID(r)
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

	// Get posts
	posts, err := h.GroupPostService.GetByGroup(groupID, currentUserID, limit, offset)
	if err != nil {
		if err.Error() == "not authorized to view posts in this group" {
			utils.RespondWithError(w, http.StatusForbidden, "Not authorized to view posts in this group")
		} else {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get group posts")
		}
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Group posts retrieved successfully", map[string]interface{}{
		"posts": posts,
	})
}

// CreateGroupPost handles creating a post in a group
func (h *Handler) CreateGroupPost(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Check if user is a member of the group
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	if !isMember {
		utils.RespondWithError(w, http.StatusForbidden, "Not a member of this group")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	// Get form values
	content := r.FormValue("content")

	// Validate content
	if content == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Content is required")
		return
	}

	// Create post
	post := &models.GroupPost{
		GroupID: groupID,
		UserID:  userID,
		Content: content,
	}

	// Check if image was uploaded
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		// Save image
		imagePath, err := utils.SaveImage(file, header, "group_posts")
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		post.Image = imagePath
	}

	// Save post
	if err := h.GroupPostService.Create(post); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create post")
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

	utils.RespondWithSuccess(w, http.StatusCreated, "Post created successfully", map[string]interface{}{
		"post": post,
	})
}

// GetGroupEvents handles retrieving events for a group
func (h *Handler) GetGroupEvents(w http.ResponseWriter, r *http.Request) {
	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get current user ID from context
	currentUserID, err := middleware.GetUserID(r)
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

	// Get events
	events, err := h.EventService.GetByGroup(groupID, currentUserID, limit, offset)
	if err != nil {
		if err.Error() == "not authorized to view events in this group" {
			utils.RespondWithError(w, http.StatusForbidden, "Not authorized to view events in this group")
		} else {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get group events")
		}
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Group events retrieved successfully", map[string]interface{}{
		"events": events,
	})
}

// CreateGroupEvent handles creating an event in a group
func (h *Handler) CreateGroupEvent(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Check if user is a member of the group
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	if !isMember {
		utils.RespondWithError(w, http.StatusForbidden, "Not a member of this group")
		return
	}

	// Parse request body
	var req CreateGroupEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Title == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Title is required")
		return
	}

	if req.StartTime == "" || req.EndTime == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Start time and end time are required")
		return
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid start time format")
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid end time format")
		return
	}

	// Create event
	event := &models.Event{
		GroupID:     groupID,
		CreatorID:   userID,
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		StartTime:   startTime,
		EndTime:     endTime,
	}

	// Save event
	if err := h.EventService.Create(event); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create event")
		return
	}

	// Get user for response
	user, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	user.Password = ""

	// Add creator to event for response
	event.Creator = user

	// Create notifications for all group members (except the creator)
	go func() {
		// Get all group members
		members, err := h.GroupMemberService.GetMembers(groupID, 1000, 0) // Get up to 1000 members
		if err != nil {
			// Log error but don't fail the request
			// TODO: Add proper logging
			return
		}

		// Also get the group creator if they're not already in the members list
		group, err := h.GroupService.GetByID(groupID, userID)
		if err != nil {
			// Log error but don't fail the request
			return
		}

		// Create a map to track unique user IDs to avoid duplicate notifications
		memberUserIDs := make(map[string]bool)
		var notifications []*models.Notification

		// Add notifications for all group members
		for _, member := range members {
			if member.UserID != userID { // Don't notify the event creator
				memberUserIDs[member.UserID] = true
				notification := &models.Notification{
					UserID:   member.UserID,
					SenderID: userID,
					Type:     models.NotificationTypeGroupEventCreated,
					Content:  "created a new event in " + group.Name,
					Data:     `{"eventId":"` + event.ID + `","groupId":"` + groupID + `","eventTitle":"` + event.Title + `"}`,
				}
				notifications = append(notifications, notification)
			}
		}

		// Add notification for group creator if they're not already included and not the event creator
		if group.CreatorID != userID && !memberUserIDs[group.CreatorID] {
			notification := &models.Notification{
				UserID:   group.CreatorID,
				SenderID: userID,
				Type:     models.NotificationTypeGroupEventCreated,
				Content:  "created a new event in " + group.Name,
				Data:     `{"eventId":"` + event.ID + `","groupId":"` + groupID + `","eventTitle":"` + event.Title + `"}`,
			}
			notifications = append(notifications, notification)
		}

		// Create all notifications in batch
		if len(notifications) > 0 {
			if err := h.NotificationService.CreateBatch(notifications); err != nil {
				// Log error but don't fail the request
				// TODO: Add proper logging
			}
		}
	}()

	utils.RespondWithSuccess(w, http.StatusCreated, "Event created successfully", map[string]interface{}{
		"event": event,
	})
}

// RespondToEvent handles responding to an event
func (h *Handler) RespondToEvent(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get event ID from URL
	vars := mux.Vars(r)
	eventID := vars["id"]

	// Parse request body
	var req EventResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate response
	if req.Response != models.EventResponseGoing &&
		req.Response != models.EventResponseMaybe &&
		req.Response != models.EventResponseNotGoing {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid response")
		return
	}

	// Check if event exists
	_, err = h.EventService.GetByID(eventID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Event not found")
		return
	}

	// Create or update response
	response := &models.EventResponse{
		EventID:  eventID,
		UserID:   userID,
		Response: req.Response,
	}

	if err := h.EventResponseService.Create(response); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to respond to event")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Response saved successfully", map[string]interface{}{
		"response": req.Response,
	})
}

// UpdateGroupEvent handles updating an event
func (h *Handler) UpdateGroupEvent(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get event ID from URL
	vars := mux.Vars(r)
	eventID := vars["id"]

	// Parse request body
	var req CreateGroupEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Title == "" || req.StartTime == "" || req.EndTime == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Title, start time, and end time are required")
		return
	}

	// Get the existing event
	existingEvent, err := h.EventService.GetByID(eventID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Event not found")
		return
	}

	// Check if user is a group admin (includes group creator)
	// Only group admins can update events, regardless of who created them
	isAdmin, err := h.GroupMemberService.IsGroupAdmin(existingEvent.GroupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check admin status")
		return
	}

	if !isAdmin {
		utils.RespondWithError(w, http.StatusForbidden, "Only group admins can update events")
		return
	}

	// Parse and validate times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid start time format")
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid end time format")
		return
	}

	// Update event
	existingEvent.Title = req.Title
	existingEvent.Description = req.Description
	existingEvent.Location = req.Location
	existingEvent.StartTime = startTime
	existingEvent.EndTime = endTime

	if err := h.EventService.Update(existingEvent); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update event")
		return
	}

	// Get updated event with creator info
	updatedEvent, err := h.EventService.GetByID(eventID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get updated event")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Event updated successfully", map[string]interface{}{
		"event": updatedEvent,
	})
}

// DeleteGroupEvent handles deleting an event
func (h *Handler) DeleteGroupEvent(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get event ID from URL
	vars := mux.Vars(r)
	eventID := vars["id"]

	// Get the existing event
	existingEvent, err := h.EventService.GetByID(eventID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Event not found")
		return
	}

	// Check authorization based on requirements:
	// - Group creator can delete anyone's event in their group
	// - Group admin can delete member events only (not creator events)
	// - Regular member can delete only their own events

	canDelete := false

	// User can delete their own event
	if existingEvent.CreatorID == userID {
		canDelete = true
	} else {
		// For events created by others, check group permissions
		group, err := h.GroupService.GetByID(existingEvent.GroupID, userID)
		if err == nil {
			if group.CreatorID == userID {
				// Group creator can delete any event in their group
				canDelete = true
			} else {
				// Check if user is a group admin
				groupMember, memberErr := h.GroupMemberService.GetByGroupAndUser(group.ID, userID)
				if memberErr == nil && groupMember.Role == models.GroupMemberRoleAdmin {
					// Group admin can delete member events only, not creator events
					if existingEvent.CreatorID != group.CreatorID {
						canDelete = true
					}
				}
			}
		}
	}

	if !canDelete {
		utils.RespondWithError(w, http.StatusForbidden, "Forbidden: You are not authorized to delete this event")
		return
	}

	// Delete event
	if err := h.EventService.Delete(eventID, userID); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete event")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Event deleted successfully", nil)
}

// GetGroupMembers handles retrieving members of a group
func (h *Handler) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get current user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check if user is a member of the group
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	if !isMember {
		utils.RespondWithError(w, http.StatusForbidden, "Not a member of this group")
		return
	}

	// Get members
	members, err := h.GroupMemberService.GetMembers(groupID, 100, 0) // Get up to 100 members
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get group members")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Group members retrieved successfully", map[string]interface{}{
		"members": members,
	})
}

// GetGroupPendingRequests handles retrieving pending join requests for a group
func (h *Handler) GetGroupPendingRequests(w http.ResponseWriter, r *http.Request) {
	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get current user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check if user is an admin of the group
	isAdmin, err := h.GroupMemberService.IsGroupAdmin(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check admin status")
		return
	}

	if !isAdmin {
		utils.RespondWithError(w, http.StatusForbidden, "Only group admins can view pending requests")
		return
	}

	// Get pending requests
	requests, err := h.GroupMemberService.GetPendingRequests(groupID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get pending requests")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Pending requests retrieved successfully", map[string]interface{}{
		"requests": requests,
	})
}

// ApproveJoinRequest handles approving a join request
func (h *Handler) ApproveJoinRequest(w http.ResponseWriter, r *http.Request) {
	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get current user ID from context
	currentUserID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get the pending member request
	member, err := h.GroupMemberService.GetByGroupAndUser(groupID, req.UserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Join request not found")
		return
	}

	if member.Status != models.GroupMemberStatusPending {
		utils.RespondWithError(w, http.StatusBadRequest, "Request is not pending")
		return
	}

	// Update status to accepted
	if err := h.GroupMemberService.UpdateStatus(member.ID, models.GroupMemberStatusAccepted); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to approve request")
		return
	}

	// Update the original group join request notification status for the current user (admin)
	if err := h.NotificationService.UpdateStatusByTypeAndSender(currentUserID, req.UserID, models.NotificationTypeGroupJoinRequest, models.NotificationStatusApproved); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
	}

	// Get group information for notification
	group, err := h.GroupService.GetByID(groupID, currentUserID)
	if err != nil {
		// Log error but don't fail the request
	} else {
		// Create notification for the user whose request was approved
		notification := &models.Notification{
			UserID:   req.UserID,
			SenderID: currentUserID,
			Type:     models.NotificationTypeGroupJoinApproved,
			Content:  "approved your request to join the group",
			Data:     `{"groupId":"` + groupID + `","groupName":"` + group.Name + `"}`,
		}

		if err := h.NotificationService.Create(notification); err != nil {
			// Log error but don't fail the request
		}
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Join request approved successfully", nil)
}

// RejectJoinRequest handles rejecting a join request
func (h *Handler) RejectJoinRequest(w http.ResponseWriter, r *http.Request) {
	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get current user ID from context
	currentUserID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check if current user is an admin of the group
	isAdmin, err := h.GroupMemberService.IsGroupAdmin(groupID, currentUserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check admin status")
		return
	}

	if !isAdmin {
		utils.RespondWithError(w, http.StatusForbidden, "Only group admins can reject join requests")
		return
	}

	// Get the pending member request
	member, err := h.GroupMemberService.GetByGroupAndUser(groupID, req.UserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Join request not found")
		return
	}

	if member.Status != models.GroupMemberStatusPending {
		utils.RespondWithError(w, http.StatusBadRequest, "Request is not pending")
		return
	}

	// Update status to rejected
	if err := h.GroupMemberService.UpdateStatus(member.ID, models.GroupMemberStatusRejected); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to reject request")
		return
	}

	// Update the original group join request notification status for the current user (admin)
	if err := h.NotificationService.UpdateStatusByTypeAndSender(currentUserID, req.UserID, models.NotificationTypeGroupJoinRequest, models.NotificationStatusRejected); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
	}

	// Get group information for notification
	group, err := h.GroupService.GetByID(groupID, currentUserID)
	if err != nil {
		// Log error but don't fail the request
	} else {
		// Create notification for the user whose request was rejected
		notification := &models.Notification{
			UserID:   req.UserID,
			SenderID: currentUserID,
			Type:     models.NotificationTypeGroupJoinRejected,
			Content:  "declined your request to join the group",
			Data:     `{"groupId":"` + groupID + `","groupName":"` + group.Name + `"}`,
		}

		if err := h.NotificationService.Create(notification); err != nil {
			// Log error but don't fail the request
		}
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Join request rejected successfully", nil)
}

// InviteToGroup handles inviting a user to a group
func (h *Handler) InviteToGroup(w http.ResponseWriter, r *http.Request) {
	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get current user ID from context
	currentUserID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Allow all group members to invite users (previously only admins could)
	// No permission check needed here as per new requirement.

	// Check if user is already a member
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, req.UserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check membership")
		return
	}

	if isMember {
		utils.RespondWithError(w, http.StatusConflict, "User is already a member of this group")
		return
	}

	// Check if there's already a pending request or invitation
	existingMember, err := h.GroupMemberService.GetByGroupAndUser(groupID, req.UserID)
	if err == nil {
		if existingMember.Status == models.GroupMemberStatusPending {
			utils.RespondWithError(w, http.StatusConflict, "User already has a pending request for this group")
			return
		}
		if existingMember.Status == models.GroupMemberStatusInvited {
			utils.RespondWithError(w, http.StatusConflict, "User already has a pending invitation for this group")
			return
		}
	}

	// Create group member with invited status (user needs to accept the invitation)
	member := &models.GroupMember{
		GroupID: groupID,
		UserID:  req.UserID,
		Role:    models.GroupMemberRoleMember,
		Status:  models.GroupMemberStatusInvited,
	}

	if err := h.GroupMemberService.Create(member); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to invite user")
		return
	}

	// Get group info for notification
	group, err := h.GroupService.GetByID(groupID, currentUserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get group info")
		return
	}

	// Create notification for invited user
	notification := &models.Notification{
		UserID:   req.UserID,
		SenderID: currentUserID,
		Type:     models.NotificationTypeGroupInvite,
		Content:  "invited you to join the group",
		Data:     `{"groupId":"` + groupID + `","groupName":"` + group.Name + `"}`,
	}

	if err := h.NotificationService.Create(notification); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
	}

	utils.RespondWithSuccess(w, http.StatusOK, "User invited successfully", nil)
}

// RespondToGroupInvitation handles accepting or declining a group invitation
func (h *Handler) RespondToGroupInvitation(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get notification ID from URL
	vars := mux.Vars(r)
	notificationID := vars["id"]

	// Parse request body
	var req struct {
		Accept bool `json:"accept"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get the notification to extract group information
	notification, err := h.NotificationService.GetByID(notificationID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Notification not found")
		return
	}

	// Verify this is a group invite notification for the current user
	if notification.UserID != userID || notification.Type != models.NotificationTypeGroupInvite {
		utils.RespondWithError(w, http.StatusForbidden, "Invalid notification")
		return
	}

	// Parse group data from notification
	var notificationData struct {
		GroupID   string `json:"groupId"`
		GroupName string `json:"groupName"`
	}
	if err := json.Unmarshal([]byte(notification.Data), &notificationData); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to parse notification data")
		return
	}

	// Get the invitation record
	invitation, err := h.GroupMemberService.GetByGroupAndUser(notificationData.GroupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Invitation not found")
		return
	}

	// Verify the invitation is still pending
	if invitation.Status != models.GroupMemberStatusInvited {
		utils.RespondWithError(w, http.StatusBadRequest, "Invitation is no longer pending")
		return
	}

	// Update the invitation status based on user's response
	var newStatus models.GroupMemberStatus
	if req.Accept {
		newStatus = models.GroupMemberStatusAccepted
	} else {
		newStatus = models.GroupMemberStatusRejected
	}

	if err := h.GroupMemberService.UpdateStatus(invitation.ID, newStatus); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update invitation status")
		return
	}

	// Update the group invitation notification status
	var notificationStatus models.NotificationStatus
	if req.Accept {
		notificationStatus = models.NotificationStatusAccepted
	} else {
		notificationStatus = models.NotificationStatusDeclined
	}

	if err := h.NotificationService.UpdateStatus(notificationID, userID, notificationStatus); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
	}

	responseMessage := "Invitation declined"
	if req.Accept {
		responseMessage = "Successfully joined the group"
	}

	utils.RespondWithSuccess(w, http.StatusOK, responseMessage, nil)
}

// LikeGroupPost handles liking a group post
func (h *Handler) LikeGroupPost(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID and post ID from URL
	vars := mux.Vars(r)
	groupID := vars["groupId"]
	postID := vars["postId"]

	// Check if user is a member of the group
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	if !isMember {
		utils.RespondWithError(w, http.StatusForbidden, "Not a member of this group")
		return
	}

	// Check if group post exists
	post, err := h.GroupPostService.GetByID(postID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Verify post belongs to the group
	if post.GroupID != groupID {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found in this group")
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
		// Get group info for notification
		group, err := h.GroupService.GetByID(groupID, userID)
		if err != nil {
			log.Printf("Error getting group for notification: %v", err)
		}

		// Prepare notification data with post content and group info
		postContent := post.Content
		if len(postContent) > 50 {
			postContent = postContent[:50] + "..."
		}

		notificationData := map[string]interface{}{
			"postId":      postID,
			"groupId":     groupID,
			"postContent": postContent,
		}
		if group != nil {
			notificationData["groupName"] = group.Name
		}
		dataJSON, _ := json.Marshal(notificationData)

		notification := &models.Notification{
			UserID:   post.UserID,
			SenderID: userID,
			Type:     "post_like",
			Content:  "liked your group post",
			Data:     string(dataJSON),
		}

		if err := h.NotificationService.Create(notification); err != nil {
			// Log error but don't fail the request
			log.Printf("Error creating notification: %v", err)
		}
	}

	// Broadcast like event via WebSocket
	likeEvent := map[string]interface{}{
		"postId":  postID,
		"groupId": groupID,
		"userId":  userID,
		"action":  "like",
	}

	message := map[string]interface{}{
		"type":    "group_post_like",
		"payload": likeEvent,
	}

	messageData, _ := json.Marshal(message)

	// Broadcast to group members
	h.Hub.Broadcast <- &websocket.Broadcast{
		RoomID:  "group_" + groupID,
		Message: messageData,
		Sender:  nil,
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Post liked successfully", nil)
}

// UnlikeGroupPost handles unliking a group post
func (h *Handler) UnlikeGroupPost(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID and post ID from URL
	vars := mux.Vars(r)
	groupID := vars["groupId"]
	postID := vars["postId"]

	// Check if user is a member of the group
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	if !isMember {
		utils.RespondWithError(w, http.StatusForbidden, "Not a member of this group")
		return
	}

	// Check if group post exists
	post, err := h.GroupPostService.GetByID(postID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Verify post belongs to the group
	if post.GroupID != groupID {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found in this group")
		return
	}

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
		"postId":  postID,
		"groupId": groupID,
		"userId":  userID,
		"action":  "unlike",
	}

	message := map[string]interface{}{
		"type":    "group_post_like",
		"payload": unlikeEvent,
	}

	messageData, _ := json.Marshal(message)

	// Broadcast to group members
	h.Hub.Broadcast <- &websocket.Broadcast{
		RoomID:  "group_" + groupID,
		Message: messageData,
		Sender:  nil,
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Post unliked successfully", nil)
}

// GetGroupPostComments handles retrieving comments for a group post
func (h *Handler) GetGroupPostComments(w http.ResponseWriter, r *http.Request) {
	// Get group ID and post ID from URL
	vars := mux.Vars(r)
	groupID := vars["groupId"]
	postID := vars["postId"]

	// Get current user ID from context
	currentUserID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check if user is a member of the group
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, currentUserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	if !isMember {
		utils.RespondWithError(w, http.StatusForbidden, "Not a member of this group")
		return
	}

	// Check if group post exists
	post, err := h.GroupPostService.GetByID(postID, currentUserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Verify post belongs to the group
	if post.GroupID != groupID {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found in this group")
		return
	}

	// Get comments
	comments, err := h.CommentService.GetCommentsByPost(postID, currentUserID, 50, 0)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get comments")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Comments retrieved successfully", map[string]interface{}{
		"comments": comments,
	})
}

// AddGroupPostComment handles adding a comment to a group post
func (h *Handler) AddGroupPostComment(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID and post ID from URL
	vars := mux.Vars(r)
	groupID := vars["groupId"]
	postID := vars["postId"]

	// Check if user is a member of the group
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	if !isMember {
		utils.RespondWithError(w, http.StatusForbidden, "Not a member of this group")
		return
	}

	// Check if group post exists
	post, err := h.GroupPostService.GetByID(postID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Verify post belongs to the group
	if post.GroupID != groupID {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found in this group")
		return
	}

	var content string
	var imagePath string

	// Check content type to determine how to parse the request
	contentType := r.Header.Get("Content-Type")

	if contentType == "application/json" {
		// Parse JSON request body (for text-only comments)
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		content = req.Content
	} else {
		// Parse multipart form (for comments with images)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Failed to parse form")
			return
		}

		// Get content from form
		content = r.FormValue("content")

		// Check if image was uploaded
		file, header, err := r.FormFile("image")
		if err == nil {
			defer file.Close()

			// Save image
			imagePath, err = utils.SaveImage(file, header, "comments")
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
	}

	// Validate content (allow empty content if image is provided)
	if content == "" && imagePath == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Content or image is required")
		return
	}

	// Create comment
	comment := &models.Comment{
		PostID:  postID,
		UserID:  userID,
		Content: content,
		Image:   imagePath,
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
			Content:  "commented on your group post",
			Data:     `{"postId":"` + postID + `","groupId":"` + groupID + `","comment":"` + content + `"}`,
		}

		if err := h.NotificationService.Create(notification); err != nil {
			// Log error but don't fail the request
			// TODO: Add proper logging
		}
	}

	// Broadcast new comment event via WebSocket
	newCommentEvent := map[string]interface{}{
		"postId":  postID,
		"groupId": groupID,
		"comment": comment,
	}

	message := map[string]interface{}{
		"type":    "group_post_comment",
		"payload": newCommentEvent,
	}

	messageData, _ := json.Marshal(message)

	// Broadcast to group members
	h.Hub.Broadcast <- &websocket.Broadcast{
		RoomID:  "group_" + groupID,
		Message: messageData,
		Sender:  nil,
	}

	utils.RespondWithSuccess(w, http.StatusCreated, "Comment added successfully", map[string]interface{}{
		"comment": comment,
	})
}

// DeleteGroupPostComment handles deleting a comment from a group post
func (h *Handler) DeleteGroupPostComment(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID, post ID, and comment ID from URL
	vars := mux.Vars(r)
	groupID := vars["groupId"]
	postID := vars["postId"]
	commentID := vars["commentId"]

	// Check if user is a member of the group
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	if !isMember {
		utils.RespondWithError(w, http.StatusForbidden, "Not a member of this group")
		return
	}

	// Check if group post exists
	post, err := h.GroupPostService.GetByID(postID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Verify post belongs to the group
	if post.GroupID != groupID {
		utils.RespondWithError(w, http.StatusNotFound, "Post not found in this group")
		return
	}

	// Get comment to check ownership
	comment, err := h.CommentService.GetByID(commentID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Comment not found")
		return
	}

	// Check if user can delete the comment (comment owner or post owner)
	if comment.UserID != userID && post.UserID != userID {
		utils.RespondWithError(w, http.StatusForbidden, "Not authorized to delete this comment")
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
		"groupId":   groupID,
		"commentId": commentID,
	}

	message := map[string]interface{}{
		"type":    "group_post_comment_delete",
		"payload": deleteCommentEvent,
	}

	messageData, _ := json.Marshal(message)

	// Broadcast to group members
	h.Hub.Broadcast <- &websocket.Broadcast{
		RoomID:  "group_" + groupID,
		Message: messageData,
		Sender:  nil,
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Comment deleted successfully", nil)
}

// GetGroupMessages handles retrieving messages for a group
func (h *Handler) GetGroupMessages(w http.ResponseWriter, r *http.Request) {
	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get current user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check if user is a member of the group
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	if !isMember {
		utils.RespondWithError(w, http.StatusForbidden, "Not a member of this group")
		return
	}

	// Get messages
	messages, err := h.MessageService.GetGroupMessages(groupID, 50, 0)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get group messages")
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Group messages retrieved successfully", map[string]interface{}{
		"messages": messages,
	})
}

// SendGroupMessage handles sending a message to a group
func (h *Handler) SendGroupMessage(w http.ResponseWriter, r *http.Request) {
	// Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["id"]

	// Get current user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Content == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Content is required")
		return
	}

	// Check if user is a member of the group
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	if !isMember {
		utils.RespondWithError(w, http.StatusForbidden, "Not a member of this group")
		return
	}

	// Create message
	message := &models.Message{
		SenderID: userID,
		GroupID:  groupID,
		Content:  req.Content,
	}

	// Save to database
	if err := h.MessageService.Create(message); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to send message")
		return
	}

	// Get user for response
	user, err := h.UserService.GetByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	user.Password = ""

	// Add sender to message for response
	message.Sender = user

	// Broadcast the message via WebSocket for real-time delivery
	roomID := "group-" + groupID
	responseMsg := map[string]interface{}{
		"roomId": roomID,
		"message": map[string]interface{}{
			"id":        message.ID,
			"content":   req.Content,
			"sender":    userID,
			"groupId":   groupID,
			"timestamp": message.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"senderInfo": map[string]interface{}{
				"id":             user.ID,
				"username":       user.Username,
				"fullName":       user.FullName,
				"profilePicture": user.ProfilePicture,
			},
		},
	}

	// Serialize the response message
	data, err := json.Marshal(map[string]interface{}{
		"type":    "new_message",
		"payload": responseMsg,
	})
	if err == nil {
		// Broadcast to the room via WebSocket
		h.Hub.Broadcast <- &websocket.Broadcast{
			RoomID:  roomID,
			Message: data,
			Sender:  nil, // No specific sender client since this is from HTTP API
		}
	}

	utils.RespondWithSuccess(w, http.StatusCreated, "Message sent successfully", map[string]interface{}{
		"message": message,
	})
}

// DeleteGroupPost handles deleting a group post
func (h *Handler) DeleteGroupPost(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get group ID and post ID from URL
	vars := mux.Vars(r)
	groupID := vars["groupId"]
	postID := vars["postId"]

	// Check if user is a group member
	isMember, err := h.GroupMemberService.IsGroupMember(groupID, userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	if !isMember {
		utils.RespondWithError(w, http.StatusForbidden, "You must be a group member to delete posts")
		return
	}

	// Delete the group post using GroupPostService which has proper permission logic
	if err := h.GroupPostService.Delete(postID, userID); err != nil {
		if err.Error() == "group post not found" {
			utils.RespondWithError(w, http.StatusNotFound, "Post not found")
		} else if err.Error() == "not authorized to delete this post" ||
			err.Error() == "admin cannot delete the group creator's post" ||
			err.Error() == "user is not an active member of this group" {
			utils.RespondWithError(w, http.StatusForbidden, err.Error())
		} else {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete post")
		}
		return
	}

	utils.RespondWithSuccess(w, http.StatusOK, "Post deleted successfully", nil)
}
