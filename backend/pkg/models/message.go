package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Message represents a chat message
type Message struct {
	ID         string     `json:"id"`
	SenderID   string     `json:"senderId"`
	ReceiverID string     `json:"receiverId,omitempty"`
	GroupID    string     `json:"groupId,omitempty"`
	Content    string     `json:"content"`
	CreatedAt  time.Time  `json:"createdAt"`
	ReadAt     *time.Time `json:"readAt,omitempty"`
	// Additional fields for API responses
	Sender *User `json:"sender,omitempty"`
}

// MessageService handles message-related operations
type MessageService struct {
	DB *sql.DB
}

// NewMessageService creates a new MessageService
func NewMessageService(db *sql.DB) *MessageService {
	return &MessageService{DB: db}
}

// Create creates a new message
func (s *MessageService) Create(message *Message) error {
	message.ID = uuid.New().String()
	message.CreatedAt = time.Now()

	// Validate that either receiverId or groupId is set, but not both
	if (message.ReceiverID == "" && message.GroupID == "") || (message.ReceiverID != "" && message.GroupID != "") {
		return errors.New("either receiverId or groupId must be set, but not both")
	}

	// Prepare SQL values - use NULL for empty strings to satisfy CHECK constraint
	var receiverID, groupID interface{}
	if message.ReceiverID != "" {
		receiverID = message.ReceiverID
		groupID = nil // Explicitly set to NULL for private messages
	} else {
		receiverID = nil // Explicitly set to NULL for group messages
		groupID = message.GroupID
	}

	_, err := s.DB.Exec(`
		INSERT INTO messages (id, sender_id, receiver_id, group_id, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, message.ID, message.SenderID, receiverID, groupID, message.Content, message.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

// GetByID retrieves a message by ID
func (s *MessageService) GetByID(id string) (*Message, error) {
	message := &Message{Sender: &User{}}
	var readAt sql.NullTime

	err := s.DB.QueryRow(`
		SELECT m.id, m.sender_id, m.receiver_id, m.group_id, m.content, m.created_at, m.read_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.id = ?
	`, id).Scan(
		&message.ID, &message.SenderID, &message.ReceiverID, &message.GroupID, &message.Content, &message.CreatedAt, &readAt,
		&message.Sender.ID, &message.Sender.Username, &message.Sender.FullName, &message.Sender.ProfilePicture,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	if readAt.Valid {
		message.ReadAt = &readAt.Time
	}

	return message, nil
}

// GetPrivateMessages retrieves private messages between two users
func (s *MessageService) GetPrivateMessages(user1ID, user2ID string, limit, offset int) ([]*Message, error) {
	rows, err := s.DB.Query(`
		SELECT m.id, m.sender_id, m.receiver_id, m.content, m.created_at, m.read_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE (m.sender_id = ? AND m.receiver_id = ?) OR (m.sender_id = ? AND m.receiver_id = ?)
		ORDER BY m.created_at DESC
		LIMIT ? OFFSET ?
	`, user1ID, user2ID, user2ID, user1ID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get private messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		message := &Message{Sender: &User{}}
		var readAt sql.NullTime

		err := rows.Scan(
			&message.ID, &message.SenderID, &message.ReceiverID, &message.Content, &message.CreatedAt, &readAt,
			&message.Sender.ID, &message.Sender.Username, &message.Sender.FullName, &message.Sender.ProfilePicture,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if readAt.Valid {
			message.ReadAt = &readAt.Time
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// GetGroupMessages retrieves messages for a group
func (s *MessageService) GetGroupMessages(groupID string, limit, offset int) ([]*Message, error) {
	rows, err := s.DB.Query(`
		SELECT m.id, m.sender_id, m.group_id, m.content, m.created_at, m.read_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.group_id = ?
		ORDER BY m.created_at DESC
		LIMIT ? OFFSET ?
	`, groupID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get group messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		message := &Message{Sender: &User{}}
		var readAt sql.NullTime

		err := rows.Scan(
			&message.ID, &message.SenderID, &message.GroupID, &message.Content, &message.CreatedAt, &readAt,
			&message.Sender.ID, &message.Sender.Username, &message.Sender.FullName, &message.Sender.ProfilePicture,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if readAt.Valid {
			message.ReadAt = &readAt.Time
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// MarkAsRead marks a message as read
func (s *MessageService) MarkAsRead(id, userID string) error {
	// Check if message is for the user
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM messages WHERE id = ? AND receiver_id = ?", id, userID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check message recipient: %w", err)
	}

	if count == 0 {
		return errors.New("message not found or not authorized")
	}

	// Mark as read
	now := time.Now()
	_, err = s.DB.Exec("UPDATE messages SET read_at = ? WHERE id = ?", now, id)
	if err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
	}

	return nil
}

// MarkAllAsRead marks all messages from a sender to a receiver as read
func (s *MessageService) MarkAllAsRead(senderID, receiverID string) error {
	now := time.Now()
	_, err := s.DB.Exec(`
		UPDATE messages
		SET read_at = ?
		WHERE sender_id = ? AND receiver_id = ? AND read_at IS NULL
	`, now, senderID, receiverID)
	if err != nil {
		return fmt.Errorf("failed to mark all messages as read: %w", err)
	}

	return nil
}

// GetUnreadCount returns the number of unread messages for a user
func (s *MessageService) GetUnreadCount(userID string) (int, error) {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM messages WHERE receiver_id = ? AND read_at IS NULL", userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// GetConversations returns a list of users the current user has conversations with
func (s *MessageService) GetConversations(userID string) ([]*User, error) {
	rows, err := s.DB.Query(`
		SELECT DISTINCT u.id, u.username, u.email, u.password, u.full_name, u.bio, u.profile_picture, u.cover_photo, u.is_private, u.created_at, u.updated_at
		FROM users u
		JOIN messages m ON (m.sender_id = u.id AND m.receiver_id = ?) OR (m.receiver_id = u.id AND m.sender_id = ?)
		WHERE u.id != ?
		ORDER BY (
			SELECT MAX(created_at)
			FROM messages
			WHERE (sender_id = u.id AND receiver_id = ?) OR (receiver_id = u.id AND sender_id = ?)
		) DESC
	`, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.Password, &user.FullName, &user.Bio, &user.ProfilePicture, &user.CoverPhoto, &user.IsPrivate, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		// Remove password from response
		user.Password = ""

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}
