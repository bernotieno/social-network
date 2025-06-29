package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Event represents a group event
type Event struct {
	ID          string    `json:"id"`
	GroupID     string    `json:"groupId"`
	CreatorID   string    `json:"creatorId"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	// Additional fields for API responses
	Creator       *User  `json:"creator,omitempty"`
	Group         *Group `json:"group,omitempty"`
	GoingCount    int    `json:"goingCount,omitempty"`
	MaybeCount    int    `json:"maybeCount,omitempty"`
	DeclinedCount int    `json:"declinedCount,omitempty"`
	UserResponse  string `json:"userResponse,omitempty"`
}

// EventService handles event-related operations
type EventService struct {
	DB *sql.DB
}

// NewEventService creates a new EventService
func NewEventService(db *sql.DB) *EventService {
	return &EventService{DB: db}
}

// Create creates a new event
func (s *EventService) Create(event *Event) error {
	event.ID = uuid.New().String()
	now := time.Now()
	event.CreatedAt = now
	event.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO events (id, group_id, creator_id, title, description, location, start_time, end_time, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.GroupID, event.CreatorID, event.Title, event.Description, event.Location, event.StartTime, event.EndTime, event.CreatedAt, event.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	return nil
}

// GetByID retrieves an event by ID
func (s *EventService) GetByID(id string, currentUserID string) (*Event, error) {
	event := &Event{Creator: &User{}, Group: &Group{}}
	var userResponse sql.NullString

	err := s.DB.QueryRow(`
		SELECT e.id, e.group_id, e.creator_id, e.title, e.description, e.location, e.start_time, e.end_time, e.created_at, e.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			g.id, g.name, g.privacy,
			(SELECT COUNT(*) FROM event_responses WHERE event_id = e.id AND response = 'going') as going_count,
			(SELECT COUNT(*) FROM event_responses WHERE event_id = e.id AND response = 'maybe') as maybe_count,
			(SELECT COUNT(*) FROM event_responses WHERE event_id = e.id AND response = 'not_going') as declined_count,
			(SELECT response FROM event_responses WHERE event_id = e.id AND user_id = ?) as user_response
		FROM events e
		JOIN users u ON e.creator_id = u.id
		JOIN groups g ON e.group_id = g.id
		WHERE e.id = ?
	`, currentUserID, id).Scan(
		&event.ID, &event.GroupID, &event.CreatorID, &event.Title, &event.Description, &event.Location, &event.StartTime, &event.EndTime, &event.CreatedAt, &event.UpdatedAt,
		&event.Creator.ID, &event.Creator.Username, &event.Creator.FullName, &event.Creator.ProfilePicture,
		&event.Group.ID, &event.Group.Name, &event.Group.Privacy,
		&event.GoingCount, &event.MaybeCount, &event.DeclinedCount, &userResponse,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("event not found")
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	if userResponse.Valid {
		event.UserResponse = userResponse.String
	}

	// Check if the current user can view this event
	if event.Group.Privacy == GroupPrivacyPrivate {
		// Check if the current user is a member of the group
		var isMember bool
		err := s.DB.QueryRow(`
			SELECT COUNT(*) > 0
			FROM group_members
			WHERE group_id = ? AND user_id = ? AND status = 'accepted'
		`, event.GroupID, currentUserID).Scan(&isMember)

		if err != nil {
			return nil, fmt.Errorf("failed to check group membership: %w", err)
		}

		if !isMember {
			return nil, errors.New("not authorized to view this event")
		}
	}

	return event, nil
}

// Update updates an event
func (s *EventService) Update(event *Event) error {
	event.UpdatedAt = time.Now()

	_, err := s.DB.Exec(`
		UPDATE events
		SET title = ?, description = ?, location = ?, start_time = ?, end_time = ?, updated_at = ?
		WHERE id = ? AND creator_id = ?
	`, event.Title, event.Description, event.Location, event.StartTime, event.EndTime, event.UpdatedAt, event.ID, event.CreatorID)

	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

// Delete deletes an event
func (s *EventService) Delete(id, userID string) error {
	// Check if user is the event creator or a group admin
	var groupID string
	var creatorID string
	err := s.DB.QueryRow(`
		SELECT group_id, creator_id
		FROM events
		WHERE id = ?
	`, id).Scan(&groupID, &creatorID)

	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("event not found")
		}
		return fmt.Errorf("failed to check event ownership: %w", err)
	}

	// Check if user is the event creator or a group admin
	if creatorID != userID {
		// Check if user is a group admin
		var isAdmin bool
		err := s.DB.QueryRow(`
			SELECT COUNT(*) > 0
			FROM group_members
			WHERE group_id = ? AND user_id = ? AND role = 'admin' AND status = 'accepted'
		`, groupID, userID).Scan(&isAdmin)

		if err != nil {
			return fmt.Errorf("failed to check admin status: %w", err)
		}

		if !isAdmin {
			return errors.New("not authorized to delete this event")
		}
	}

	// Delete the event
	_, err = s.DB.Exec("DELETE FROM events WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

// GetByGroup retrieves events for a group
func (s *EventService) GetByGroup(groupID, currentUserID string, limit, offset int) ([]*Event, error) {
	// Check if the current user can view events in this group
	var groupPrivacy GroupPrivacy
	err := s.DB.QueryRow("SELECT privacy FROM groups WHERE id = ?", groupID).Scan(&groupPrivacy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("group not found")
		}
		return nil, fmt.Errorf("failed to check group privacy: %w", err)
	}

	if groupPrivacy == GroupPrivacyPrivate {
		// Check if the current user is a member of the group
		var isMember bool
		err := s.DB.QueryRow(`
			SELECT COUNT(*) > 0
			FROM group_members
			WHERE group_id = ? AND user_id = ? AND status = 'accepted'
		`, groupID, currentUserID).Scan(&isMember)

		if err != nil {
			return nil, fmt.Errorf("failed to check group membership: %w", err)
		}

		if !isMember {
			return nil, errors.New("not authorized to view events in this group")
		}
	}

	// Get events
	rows, err := s.DB.Query(`
		SELECT e.id, e.group_id, e.creator_id, e.title, e.description, e.location, e.start_time, e.end_time, e.created_at, e.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			(SELECT COUNT(*) FROM event_responses WHERE event_id = e.id AND response = 'going') as going_count,
			(SELECT COUNT(*) FROM event_responses WHERE event_id = e.id AND response = 'maybe') as maybe_count,
			(SELECT COUNT(*) FROM event_responses WHERE event_id = e.id AND response = 'not_going') as declined_count,
			(SELECT response FROM event_responses WHERE event_id = e.id AND user_id = ?) as user_response
		FROM events e
		JOIN users u ON e.creator_id = u.id
		WHERE e.group_id = ?
		ORDER BY e.start_time ASC
		LIMIT ? OFFSET ?
	`, currentUserID, groupID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to get group events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{Creator: &User{}}
		var userResponse sql.NullString

		err := rows.Scan(
			&event.ID, &event.GroupID, &event.CreatorID, &event.Title, &event.Description, &event.Location, &event.StartTime, &event.EndTime, &event.CreatedAt, &event.UpdatedAt,
			&event.Creator.ID, &event.Creator.Username, &event.Creator.FullName, &event.Creator.ProfilePicture,
			&event.GoingCount, &event.MaybeCount, &event.DeclinedCount, &userResponse,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if userResponse.Valid {
			event.UserResponse = userResponse.String
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}
