package websocket

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients by room
	Rooms map[string]map[*Client]bool

	// Inbound messages to broadcast
	Broadcast chan *Broadcast

	// Register requests from clients
	Register chan *Registration

	// Unregister requests from clients
	Unregister chan *Client
}

// Broadcast represents a message to be broadcast to a room
type Broadcast struct {
	RoomID  string
	Message []byte
	Sender  *Client
}

// Registration represents a client registration to a room
type Registration struct {
	Client *Client
	RoomID string
}

// NewHub creates a new hub
func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[string]map[*Client]bool),
		Broadcast:  make(chan *Broadcast),
		Register:   make(chan *Registration),
		Unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case registration := <-h.Register:
			// Create the room if it doesn't exist
			if _, ok := h.Rooms[registration.RoomID]; !ok {
				h.Rooms[registration.RoomID] = make(map[*Client]bool)
			}
			// Add the client to the room
			h.Rooms[registration.RoomID][registration.Client] = true

		case client := <-h.Unregister:
			// Remove the client from all rooms
			for roomID, room := range h.Rooms {
				if _, ok := room[client]; ok {
					delete(room, client)
					// Delete the room if it's empty
					if len(room) == 0 {
						delete(h.Rooms, roomID)
					}
				}
			}
			// Close the client's send channel
			close(client.Send)

		case broadcast := <-h.Broadcast:
			// Get the room
			room, ok := h.Rooms[broadcast.RoomID]
			if !ok {
				continue
			}

			// Broadcast the message to all clients in the room
			for client := range room {
				// Don't send the message back to the sender (unless sender is nil, meaning it's from HTTP API)
				if broadcast.Sender != nil && client == broadcast.Sender {
					continue
				}

				select {
				case client.Send <- broadcast.Message:
				default:
					// Client's send buffer is full, remove them
					close(client.Send)
					delete(room, client)
					// Delete the room if it's empty
					if len(room) == 0 {
						delete(h.Rooms, broadcast.RoomID)
					}
				}
			}
		}
	}
}
