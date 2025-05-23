package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeFollowRequest   NotificationType = "follow_request"
	NotificationTypeFollowAccepted  NotificationType = "follow_accepted"
	NotificationTypeNewFollower     NotificationType = "new_follower"
	NotificationTypePostLike        NotificationType = "post_like"
	NotificationTypePostComment     NotificationType = "post_comment"
	NotificationTypeGroupInvite     NotificationType = "group_invite"
	NotificationTypeGroupJoinRequest NotificationType = "group_join_request"
	NotificationTypeEventInvite     NotificationType = "event_invite"
)

// Notification represents a notification
type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"userId"`
	SenderID  string           `json:"senderId"`
	Type      NotificationType `json:"type"`
	Content   string           `json:"content"`
	Data      string           `json:"data,omitempty"`
	ReadAt    *time.Time       `json:"readAt,omitempty"`
	CreatedAt time.Time        `json:"createdAt"`
	// Additional fields for API responses
	Sender *User `json:"sender,omitempty"`
}

// NotificationService handles notification-related operations
type NotificationService struct {
	DB *sql.DB
}

// NewNotificationService creates a new NotificationService
func NewNotificationService(db *sql.DB) *NotificationService {
	return &NotificationService{DB: db}
}

// Create creates a new notification
func (s *NotificationService) Create(notification *Notification) error {
	notification.ID = uuid.New().String()
	notification.CreatedAt = time.Now()

	_, err := s.DB.Exec(`
		INSERT INTO notifications (id, user_id, sender_id, type, content, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, notification.ID, notification.UserID, notification.SenderID, notification.Type, notification.Content, notification.Data, notification.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

// GetByID retrieves a notification by ID
func (s *NotificationService) GetByID(id string) (*Notification, error) {
	notification := &Notification{Sender: &User{}}
	var readAt sql.NullTime

	err := s.DB.QueryRow(`
		SELECT n.id, n.user_id, n.sender_id, n.type, n.content, n.data, n.read_at, n.created_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM notifications n
		JOIN users u ON n.sender_id = u.id
		WHERE n.id = ?
	`, id).Scan(
		&notification.ID, &notification.UserID, &notification.SenderID, &notification.Type, &notification.Content, &notification.Data, &readAt, &notification.CreatedAt,
		&notification.Sender.ID, &notification.Sender.Username, &notification.Sender.FullName, &notification.Sender.ProfilePicture,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("notification not found")
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	if readAt.Valid {
		notification.ReadAt = &readAt.Time
	}

	return notification, nil
}

// GetByUser retrieves notifications for a user
func (s *NotificationService) GetByUser(userID string, limit, offset int) ([]*Notification, error) {
	rows, err := s.DB.Query(`
		SELECT n.id, n.user_id, n.sender_id, n.type, n.content, n.data, n.read_at, n.created_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM notifications n
		JOIN users u ON n.sender_id = u.id
		WHERE n.user_id = ?
		ORDER BY n.created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		notification := &Notification{Sender: &User{}}
		var readAt sql.NullTime

		err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.SenderID, &notification.Type, &notification.Content, &notification.Data, &readAt, &notification.CreatedAt,
			&notification.Sender.ID, &notification.Sender.Username, &notification.Sender.FullName, &notification.Sender.ProfilePicture,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}

		if readAt.Valid {
			notification.ReadAt = &readAt.Time
		}

		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notifications: %w", err)
	}

	return notifications, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(id, userID string) error {
	// Check if notification belongs to user
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM notifications WHERE id = ? AND user_id = ?", id, userID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check notification ownership: %w", err)
	}

	if count == 0 {
		return errors.New("notification not found or not authorized")
	}

	// Mark as read
	now := time.Now()
	_, err = s.DB.Exec("UPDATE notifications SET read_at = ? WHERE id = ?", now, id)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}

// MarkAllAsRead marks all notifications for a user as read
func (s *NotificationService) MarkAllAsRead(userID string) error {
	now := time.Now()
	_, err := s.DB.Exec("UPDATE notifications SET read_at = ? WHERE user_id = ? AND read_at IS NULL", now, userID)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	return nil
}

// GetUnreadCount returns the number of unread notifications for a user
func (s *NotificationService) GetUnreadCount(userID string) (int, error) {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL", userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}
