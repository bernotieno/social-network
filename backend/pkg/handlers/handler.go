package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/bernaotieno/social-network/backend/pkg/middleware"
	"github.com/bernaotieno/social-network/backend/pkg/models"
	"github.com/bernaotieno/social-network/backend/pkg/websocket"
	"github.com/gorilla/mux"
)

// MessageServiceAdapter adapts the models.MessageService to the websocket.MessageService interface
type MessageServiceAdapter struct {
	service *models.MessageService
}

// Create adapts the Create method to match the websocket interface
func (a *MessageServiceAdapter) Create(dbMessage *websocket.DBMessage) error {
	message := &models.Message{
		SenderID:   dbMessage.SenderID,
		ReceiverID: dbMessage.ReceiverID,
		Content:    dbMessage.Content,
	}
	return a.service.Create(message)
}

// Handler contains all the HTTP handlers for the API
type Handler struct {
	DB                   *sql.DB
	Hub                  *websocket.Hub
	UserService          *models.UserService
	SessionService       *models.SessionService
	FollowService        *models.FollowService
	PostService          *models.PostService
	CommentService       *models.CommentService
	LikeService          *models.LikeService
	GroupService         *models.GroupService
	GroupMemberService   *models.GroupMemberService
	GroupPostService     *models.GroupPostService
	EventService         *models.EventService
	EventResponseService *models.EventResponseService
	MessageService       *models.MessageService
	NotificationService  *models.NotificationService
	Upgrader             websocket.Upgrader
}

// NewHandler creates a new Handler
func NewHandler(db *sql.DB, hub *websocket.Hub) *Handler {
	return &Handler{
		DB:                   db,
		Hub:                  hub,
		UserService:          models.NewUserService(db),
		SessionService:       models.NewSessionService(db),
		FollowService:        models.NewFollowService(db),
		PostService:          models.NewPostService(db),
		CommentService:       models.NewCommentService(db),
		LikeService:          models.NewLikeService(db),
		GroupService:         models.NewGroupService(db),
		GroupMemberService:   models.NewGroupMemberService(db),
		GroupPostService:     models.NewGroupPostService(db),
		EventService:         models.NewEventService(db),
		EventResponseService: models.NewEventResponseService(db),
		MessageService:       models.NewMessageService(db),
		NotificationService:  models.NewNotificationService(db),
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in development
				return true
			},
		},
	}
}

// HandleWebSocket handles WebSocket connections
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("HandleWebSocket: Starting WebSocket upgrade for user")

	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}
	log.Printf("HandleWebSocket: WebSocket upgrade successful")

	// Get the user ID from the context
	userID, err := middleware.GetUserID(r)
	if err != nil {
		log.Printf("HandleWebSocket: User ID not found in context: %v", err)
		conn.Close()
		return
	}
	log.Printf("HandleWebSocket: User ID found: %s", userID)

	// Create a new client with message service adapter
	messageAdapter := &MessageServiceAdapter{service: h.MessageService}
	client := websocket.NewClient(h.Hub, conn, userID, messageAdapter)
	log.Printf("HandleWebSocket: Created WebSocket client for user %s", userID)

	// Register the client with the hub
	registration := &websocket.Registration{
		Client: client,
		RoomID: "", // Default room
	}
	h.Hub.Register <- registration
	log.Printf("HandleWebSocket: Registered client with hub")

	// Start the client's read and write pumps
	go client.WritePump()
	go client.ReadPump()
	log.Printf("HandleWebSocket: Started read and write pumps for user %s", userID)
}

// SendMessage handles sending a message via HTTP API
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context using the middleware helper
	userID, err := middleware.GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req struct {
		ReceiverID string `json:"receiverId"`
		Content    string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ReceiverID == "" || req.Content == "" {
		http.Error(w, "ReceiverID and content are required", http.StatusBadRequest)
		return
	}

	// Create message
	message := &models.Message{
		SenderID:   userID,
		ReceiverID: req.ReceiverID,
		Content:    req.Content,
	}

	// Save to database
	if err := h.MessageService.Create(message); err != nil {
		log.Printf("Error creating message: %v", err)
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	// Also broadcast the message via WebSocket for real-time delivery
	roomID := generateRoomID(userID, req.ReceiverID)
	responseMsg := map[string]interface{}{
		"roomId": roomID,
		"message": map[string]interface{}{
			"content":   req.Content,
			"sender":    userID,
			"timestamp": message.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	// Serialize the response message
	data, err := json.Marshal(map[string]interface{}{
		"type":    "new_message",
		"payload": responseMsg,
	})
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		// Continue even if WebSocket broadcast fails
	} else {
		// Broadcast to the room via WebSocket
		h.Hub.Broadcast <- &websocket.Broadcast{
			RoomID:  roomID,
			Message: data,
			Sender:  nil, // No specific sender client since this is from HTTP API
		}
		log.Printf("Message broadcasted via WebSocket to room %s", roomID)
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Message sent successfully",
	})
}

// GetMessages handles getting messages between two users
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context using the middleware helper
	userID, err := middleware.GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get other user ID from URL
	vars := mux.Vars(r)
	otherUserID := vars["userId"]
	if otherUserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get messages between the two users (limit to 50 messages, no offset)
	messages, err := h.MessageService.GetPrivateMessages(userID, otherUserID, 50, 0)
	if err != nil {
		log.Printf("Error getting messages: %v", err)
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	// Return messages
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"messages": messages,
	})
}

// generateRoomID creates a consistent room ID for two users
func generateRoomID(userID1, userID2 string) string {
	users := []string{userID1, userID2}
	sort.Strings(users)
	return strings.Join(users, "-")
}
