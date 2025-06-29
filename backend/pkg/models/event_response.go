package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EventResponseType represents the type of response to an event
type EventResponseType string

const (
	EventResponseGoing    EventResponseType = "going"
	EventResponseMaybe    EventResponseType = "maybe"
	EventResponseNotGoing EventResponseType = "not_going"
)

// EventResponse represents a user's response to an event
type EventResponse struct {
	ID        string            `json:"id"`
	EventID   string            `json:"eventId"`
	UserID    string            `json:"userId"`
	Response  EventResponseType `json:"response"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
	// Additional fields for API responses
	User  *User  `json:"user,omitempty"`
	Event *Event `json:"event,omitempty"`
}

// EventResponseService handles event response-related operations
type EventResponseService struct {
	DB *sql.DB
}

// NewEventResponseService creates a new EventResponseService
func NewEventResponseService(db *sql.DB) *EventResponseService {
	return &EventResponseService{DB: db}
}

// Create creates a new event response
func (s *EventResponseService) Create(response *EventResponse) error {
	// Check if a response already exists
	var existingID string
	err := s.DB.QueryRow("SELECT id FROM event_responses WHERE event_id = ? AND user_id = ?", response.EventID, response.UserID).Scan(&existingID)

	if err == nil {
		// Update existing response
		response.ID = existingID
		response.UpdatedAt = time.Now()

		_, err := s.DB.Exec(`
			UPDATE event_responses
			SET response = ?, updated_at = ?
			WHERE id = ?
		`, response.Response, response.UpdatedAt, response.ID)

		if err != nil {
			return fmt.Errorf("failed to update event response: %w", err)
		}

		return nil
	}

	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing response: %w", err)
	}

	// Create new response
	response.ID = uuid.New().String()
	now := time.Now()
	response.CreatedAt = now
	response.UpdatedAt = now

	_, err = s.DB.Exec(`
		INSERT INTO event_responses (id, event_id, user_id, response, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, response.ID, response.EventID, response.UserID, response.Response, response.CreatedAt, response.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create event response: %w", err)
	}

	return nil
}

// GetByID retrieves an event response by ID
func (s *EventResponseService) GetByID(id string) (*EventResponse, error) {
	response := &EventResponse{User: &User{}, Event: &Event{}}
	err := s.DB.QueryRow(`
		SELECT er.id, er.event_id, er.user_id, er.response, er.created_at, er.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			e.id, e.title, e.start_time
		FROM event_responses er
		JOIN users u ON er.user_id = u.id
		JOIN events e ON er.event_id = e.id
		WHERE er.id = ?
	`, id).Scan(
		&response.ID, &response.EventID, &response.UserID, &response.Response, &response.CreatedAt, &response.UpdatedAt,
		&response.User.ID, &response.User.Username, &response.User.FullName, &response.User.ProfilePicture,
		&response.Event.ID, &response.Event.Title, &response.Event.StartTime,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("event response not found")
		}
		return nil, fmt.Errorf("failed to get event response: %w", err)
	}

	return response, nil
}

// GetByEventAndUser retrieves an event response by event ID and user ID
func (s *EventResponseService) GetByEventAndUser(eventID, userID string) (*EventResponse, error) {
	response := &EventResponse{}
	err := s.DB.QueryRow(`
		SELECT id, event_id, user_id, response, created_at, updated_at
		FROM event_responses
		WHERE event_id = ? AND user_id = ?
	`, eventID, userID).Scan(
		&response.ID, &response.EventID, &response.UserID, &response.Response, &response.CreatedAt, &response.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("event response not found")
		}
		return nil, fmt.Errorf("failed to get event response: %w", err)
	}

	return response, nil
}

// Delete deletes an event response
func (s *EventResponseService) Delete(eventID, userID string) error {
	_, err := s.DB.Exec("DELETE FROM event_responses WHERE event_id = ? AND user_id = ?", eventID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete event response: %w", err)
	}

	return nil
}

// GetResponsesByEvent retrieves responses for an event
func (s *EventResponseService) GetResponsesByEvent(eventID string, responseType EventResponseType) ([]*EventResponse, error) {
	rows, err := s.DB.Query(`
		SELECT er.id, er.event_id, er.user_id, er.response, er.created_at, er.updated_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM event_responses er
		JOIN users u ON er.user_id = u.id
		WHERE er.event_id = ? AND er.response = ?
		ORDER BY er.created_at ASC
	`, eventID, responseType)

	if err != nil {
		return nil, fmt.Errorf("failed to get event responses: %w", err)
	}
	defer rows.Close()

	var responses []*EventResponse
	for rows.Next() {
		response := &EventResponse{User: &User{}}
		err := rows.Scan(
			&response.ID, &response.EventID, &response.UserID, &response.Response, &response.CreatedAt, &response.UpdatedAt,
			&response.User.ID, &response.User.Username, &response.User.FullName, &response.User.ProfilePicture,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event response: %w", err)
		}
		responses = append(responses, response)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event responses: %w", err)
	}

	return responses, nil
}

// GetResponseCounts returns the counts of each response type for an event
func (s *EventResponseService) GetResponseCounts(eventID string) (map[EventResponseType]int, error) {
	rows, err := s.DB.Query(`
		SELECT response, COUNT(*) as count
		FROM event_responses
		WHERE event_id = ?
		GROUP BY response
	`, eventID)

	if err != nil {
		return nil, fmt.Errorf("failed to get response counts: %w", err)
	}
	defer rows.Close()

	counts := map[EventResponseType]int{
		EventResponseGoing:    0,
		EventResponseMaybe:    0,
		EventResponseNotGoing: 0,
	}

	for rows.Next() {
		var responseType EventResponseType
		var count int
		err := rows.Scan(&responseType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan response count: %w", err)
		}
		counts[responseType] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating response counts: %w", err)
	}

	return counts, nil
}
