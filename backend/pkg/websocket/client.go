package websocket

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 10000
)

// Message represents a WebSocket message
type Message struct {
	Type     string      `json:"type"`
	RoomID   string      `json:"roomId,omitempty"`
	Content  interface{} `json:"content"`
	SenderID string      `json:"senderId"`
}

// DBMessage represents a message for database storage
type DBMessage struct {
	SenderID   string
	ReceiverID string
	Content    string
}

// MessageService interface for saving messages
type MessageService interface {
	Create(message *DBMessage) error
}

// Client represents a connected WebSocket client
type Client struct {
	Hub            *Hub
	Conn           *websocket.Conn
	Send           chan []byte
	UserID         string
	Rooms          map[string]bool
	MessageService MessageService
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID string, messageService MessageService) *Client {
	return &Client{
		Hub:            hub,
		Conn:           conn,
		Send:           make(chan []byte, 256),
		UserID:         userID,
		Rooms:          make(map[string]bool),
		MessageService: messageService,
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Parse the raw message first to handle payload structure
		var rawMsg map[string]interface{}
		if err := json.Unmarshal(message, &rawMsg); err != nil {
			log.Printf("error parsing raw message: %v", err)
			continue
		}

		// Extract the actual message from payload if it exists
		var msg Message
		if payload, ok := rawMsg["payload"].(map[string]interface{}); ok {
			msg.Type = rawMsg["type"].(string)
			msg.Content = payload
			if roomId, ok := payload["roomId"].(string); ok {
				msg.RoomID = roomId
			}
		} else {
			// Fallback to direct parsing
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("error parsing message: %v", err)
				continue
			}
		}

		// Set the sender ID
		msg.SenderID = c.UserID

		// Handle different message types
		switch msg.Type {
		case "join_room":
			// Join a chat room
			var roomID string
			if roomIDStr, ok := msg.Content.(string); ok {
				roomID = roomIDStr
			} else if contentMap, ok := msg.Content.(map[string]interface{}); ok {
				if roomIDStr, ok := contentMap["roomId"].(string); ok {
					roomID = roomIDStr
				}
			}
			if roomID != "" {
				c.JoinRoom(roomID)
			} else {
				log.Printf("invalid room ID in join_room message")
			}
		case "leave_room":
			// Leave a chat room
			var roomID string
			if roomIDStr, ok := msg.Content.(string); ok {
				roomID = roomIDStr
			} else if contentMap, ok := msg.Content.(map[string]interface{}); ok {
				if roomIDStr, ok := contentMap["roomId"].(string); ok {
					roomID = roomIDStr
				}
			}
			if roomID != "" {
				c.LeaveRoom(roomID)
			} else {
				log.Printf("invalid room ID in leave_room message")
			}
		case "chat_message":
			// Handle chat message
			if msg.RoomID != "" && c.MessageService != nil {
				// Extract message content
				var messageContent map[string]interface{}
				if content, ok := msg.Content.(map[string]interface{}); ok {
					messageContent = content
				} else {
					log.Printf("invalid message content format")
					continue
				}

				// Parse room ID to get receiver ID (format: "userId1-userId2")
				roomParts := strings.Split(msg.RoomID, "-")
				if len(roomParts) != 2 {
					log.Printf("invalid room ID format: %s", msg.RoomID)
					continue
				}

				// Determine receiver ID (the other user in the room)
				var receiverID string
				if roomParts[0] == c.UserID {
					receiverID = roomParts[1]
				} else {
					receiverID = roomParts[0]
				}

				// Extract content from the message
				var content string
				if contentStr, ok := messageContent["content"].(string); ok {
					content = contentStr
				} else if contentObj, ok := messageContent["content"].(map[string]interface{}); ok {
					// Handle nested content object
					if contentStr, ok := contentObj["content"].(string); ok {
						content = contentStr
					} else {
						log.Printf("nested message content is not a string")
						continue
					}
				} else {
					log.Printf("message content is not a string or object")
					continue
				}

				// Create message object for database
				dbMessage := &DBMessage{
					SenderID:   c.UserID,
					ReceiverID: receiverID,
					Content:    content,
				}

				// Save message to database
				if err := c.MessageService.Create(dbMessage); err != nil {
					log.Printf("error saving message to database: %v", err)
					// Continue with broadcast even if database save fails
				}

				// Create response message for broadcast
				responseMsg := map[string]interface{}{
					"roomId": msg.RoomID,
					"message": map[string]interface{}{
						"content":   content,
						"sender":    c.UserID,
						"timestamp": time.Now().Format(time.RFC3339),
					},
				}

				// Serialize the response message
				data, err := json.Marshal(map[string]interface{}{
					"type":    "new_message",
					"payload": responseMsg,
				})
				if err != nil {
					log.Printf("error marshaling response message: %v", err)
					continue
				}

				// Broadcast to the room
				c.Hub.Broadcast <- &Broadcast{
					RoomID:  msg.RoomID,
					Message: data,
					Sender:  c,
				}
			}
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// JoinRoom adds the client to a room
func (c *Client) JoinRoom(roomID string) {
	log.Printf("Client %s joining room %s", c.UserID, roomID)

	// Leave all previous rooms first
	for oldRoomID := range c.Rooms {
		if oldRoomID != roomID {
			log.Printf("Client %s leaving previous room %s", c.UserID, oldRoomID)
			delete(c.Rooms, oldRoomID)
		}
	}

	// Join the new room
	c.Hub.Register <- &Registration{
		Client: c,
		RoomID: roomID,
	}
	c.Rooms[roomID] = true
	log.Printf("Client %s successfully joined room %s", c.UserID, roomID)
}

// LeaveRoom removes the client from a room
func (c *Client) LeaveRoom(roomID string) {
	c.Hub.Unregister <- c
	delete(c.Rooms, roomID)
}
