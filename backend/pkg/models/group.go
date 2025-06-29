package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GroupPrivacy represents the privacy level of a group
type GroupPrivacy string

const (
	GroupPrivacyPublic  GroupPrivacy = "public"
	GroupPrivacyPrivate GroupPrivacy = "private"
)

// GroupMemberRole represents the role of a user within a group
type GroupMemberRole string

const (
	GroupMemberRoleCreator GroupMemberRole = "creator"
	GroupMemberRoleAdmin   GroupMemberRole = "admin"
	GroupMemberRoleMember  GroupMemberRole = "member"
)

// GroupMemberStatus represents the status of a user's membership in a group
type GroupMemberStatus string

const (
	GroupMemberStatusPending  GroupMemberStatus = "pending"
	GroupMemberStatusAccepted GroupMemberStatus = "accepted"
	GroupMemberStatusRejected GroupMemberStatus = "rejected"
	GroupMemberStatusInvited  GroupMemberStatus = "invited"
)

// Group represents a group
type Group struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	CreatorID   string       `json:"creatorId"`
	CoverPhoto  string       `json:"coverPhoto,omitempty"`
	Privacy     GroupPrivacy `json:"privacy"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	// Additional fields for API responses
	Creator       *User  `json:"creator,omitempty"`
	MembersCount  int    `json:"membersCount,omitempty"`
	IsJoined      bool   `json:"isJoined"`
	IsAdmin       bool   `json:"isAdmin"`
	RequestStatus string `json:"requestStatus,omitempty"` // pending, accepted, rejected, none
}

// GroupService handles group-related operations
type GroupService struct {
	DB *sql.DB
}

// NewGroupService creates a new GroupService
func NewGroupService(db *sql.DB) *GroupService {
	return &GroupService{DB: db}
}

// Create creates a new group
func (s *GroupService) Create(group *Group) error {
	group.ID = uuid.New().String()
	now := time.Now()
	group.CreatedAt = now
	group.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO groups (id, name, description, creator_id, cover_photo, privacy, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, group.ID, group.Name, group.Description, group.CreatorID, group.CoverPhoto, group.Privacy, group.CreatedAt, group.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	// Add the creator as a member with 'creator' role and 'accepted' status
	_, err = s.DB.Exec(`
		INSERT INTO group_members (id, group_id, user_id, role, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, uuid.New().String(), group.ID, group.CreatorID, GroupMemberRoleCreator, GroupMemberStatusAccepted, now, now)
	if err != nil {
		return fmt.Errorf("failed to add group creator as member: %w", err)
	}

	return nil
}

// GetByID retrieves a group by ID
func (s *GroupService) GetByID(id string, currentUserID string) (*Group, error) {
	group := &Group{Creator: &User{}}
	var requestStatus sql.NullString

	err := s.DB.QueryRow(`
		SELECT g.id, g.name, g.description, g.creator_id, g.cover_photo, g.privacy, g.created_at, g.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			(SELECT COUNT(*) FROM group_members WHERE group_id = g.id AND status = 'accepted') as members_count,
			(g.creator_id = ? OR (SELECT COUNT(*) FROM group_members WHERE group_id = g.id AND user_id = ? AND status = 'accepted') > 0) as is_joined,
			(SELECT status FROM group_members WHERE group_id = g.id AND user_id = ? LIMIT 1) as request_status,
			(SELECT COUNT(*) FROM group_members WHERE group_id = g.id AND user_id = ? AND (role = 'admin' OR role = 'creator')) > 0 as is_admin
		FROM groups g
		JOIN users u ON g.creator_id = u.id
		WHERE g.id = ?
	`, currentUserID, currentUserID, currentUserID, currentUserID, id).Scan(
		&group.ID, &group.Name, &group.Description, &group.CreatorID, &group.CoverPhoto, &group.Privacy, &group.CreatedAt, &group.UpdatedAt,
		&group.Creator.ID, &group.Creator.Username, &group.Creator.FullName, &group.Creator.ProfilePicture,
		&group.MembersCount, &group.IsJoined, &requestStatus, &group.IsAdmin,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("group not found")
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	// Set request status
	if requestStatus.Valid {
		group.RequestStatus = requestStatus.String
	} else {
		group.RequestStatus = "none"
	}

	// Check if the user can view this group
	if group.Privacy == GroupPrivacyPrivate && !group.IsJoined && group.CreatorID != currentUserID {
		return nil, errors.New("not authorized to view this group")
	}

	return group, nil
}

// Update updates a group
func (s *GroupService) Update(group *Group) error {
	group.UpdatedAt = time.Now()

	_, err := s.DB.Exec(`
		UPDATE groups
		SET name = ?, description = ?, cover_photo = ?, privacy = ?, updated_at = ?
		WHERE id = ? AND creator_id = ?
	`, group.Name, group.Description, group.CoverPhoto, group.Privacy, group.UpdatedAt, group.ID, group.CreatorID)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	return nil
}

// Delete deletes a group
func (s *GroupService) Delete(id, userID string) error {
	// Check if user is the creator
	var creatorID string
	err := s.DB.QueryRow("SELECT creator_id FROM groups WHERE id = ?", id).Scan(&creatorID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("group not found")
		}
		return fmt.Errorf("failed to check group ownership: %w", err)
	}

	if creatorID != userID {
		return errors.New("not authorized to delete this group")
	}

	// Delete the group
	_, err = s.DB.Exec("DELETE FROM groups WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

// GetGroups retrieves groups with optional filtering
func (s *GroupService) GetGroups(currentUserID string, limit, offset int) ([]*Group, error) {
	rows, err := s.DB.Query(`
		SELECT g.id, g.name, g.description, g.creator_id, g.cover_photo, g.privacy, g.created_at, g.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			(SELECT COUNT(*) FROM group_members WHERE group_id = g.id AND status = 'accepted') as members_count,
			(g.creator_id = ? OR (SELECT COUNT(*) FROM group_members WHERE group_id = g.id AND user_id = ? AND status = 'accepted') > 0) as is_joined,
			(SELECT status FROM group_members WHERE group_id = g.id AND user_id = ? LIMIT 1) as request_status
		FROM groups g
		JOIN users u ON g.creator_id = u.id
		ORDER BY g.created_at DESC
		LIMIT ? OFFSET ?
	`, currentUserID, currentUserID, currentUserID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}
	defer rows.Close()

	var groups []*Group
	for rows.Next() {
		group := &Group{Creator: &User{}}
		var requestStatus sql.NullString
		err := rows.Scan(
			&group.ID, &group.Name, &group.Description, &group.CreatorID, &group.CoverPhoto, &group.Privacy, &group.CreatedAt, &group.UpdatedAt,
			&group.Creator.ID, &group.Creator.Username, &group.Creator.FullName, &group.Creator.ProfilePicture,
			&group.MembersCount, &group.IsJoined, &requestStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}

		// Set request status
		if requestStatus.Valid {
			group.RequestStatus = requestStatus.String
		} else {
			group.RequestStatus = "none"
		}

		groups = append(groups, group)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating rows: %w", err)
	}

	return groups, nil
}

// SearchGroups searches groups by name or description
func (s *GroupService) SearchGroups(query string, currentUserID string, limit, offset int) ([]*Group, error) {
	rows, err := s.DB.Query(`
		SELECT g.id, g.name, g.description, g.creator_id, g.cover_photo, g.privacy, g.created_at, g.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			(SELECT COUNT(*) FROM group_members WHERE group_id = g.id AND status = 'accepted') as members_count,
			(g.creator_id = ? OR (SELECT COUNT(*) FROM group_members WHERE group_id = g.id AND user_id = ? AND status = 'accepted') > 0) as is_joined,
			(SELECT status FROM group_members WHERE group_id = g.id AND user_id = ? LIMIT 1) as request_status
		FROM groups g
		JOIN users u ON g.creator_id = u.id
		WHERE (g.name LIKE ? OR g.description LIKE ?)
		ORDER BY g.created_at DESC
		LIMIT ? OFFSET ?
	`, currentUserID, currentUserID, currentUserID, "%"+query+"%", "%"+query+"%", limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to search groups: %w", err)
	}
	defer rows.Close()

	var groups []*Group
	for rows.Next() {
		group := &Group{Creator: &User{}}
		var requestStatus sql.NullString
		err := rows.Scan(
			&group.ID, &group.Name, &group.Description, &group.CreatorID, &group.CoverPhoto, &group.Privacy, &group.CreatedAt, &group.UpdatedAt,
			&group.Creator.ID, &group.Creator.Username, &group.Creator.FullName, &group.Creator.ProfilePicture,
			&group.MembersCount, &group.IsJoined, &requestStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}

		// Set request status
		if requestStatus.Valid {
			group.RequestStatus = requestStatus.String
		} else {
			group.RequestStatus = "none"
		}

		groups = append(groups, group)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating rows: %w", err)
	}

	return groups, nil
}

// GetUserGroups retrieves groups a user is a member of
func (s *GroupService) GetUserGroups(userID string, limit, offset int) ([]*Group, error) {
	rows, err := s.DB.Query(`
		SELECT g.id, g.name, g.description, g.creator_id, g.cover_photo, g.privacy, g.created_at, g.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			(SELECT COUNT(*) FROM group_members WHERE group_id = g.id AND status = 'accepted') as members_count,
			true as is_joined
		FROM groups g
		JOIN users u ON g.creator_id = u.id
		JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = ? AND gm.status = 'accepted'
		ORDER BY g.created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}
	defer rows.Close()

	var groups []*Group
	for rows.Next() {
		group := &Group{Creator: &User{}}
		err := rows.Scan(
			&group.ID, &group.Name, &group.Description, &group.CreatorID, &group.CoverPhoto, &group.Privacy, &group.CreatedAt, &group.UpdatedAt,
			&group.Creator.ID, &group.Creator.Username, &group.Creator.FullName, &group.Creator.ProfilePicture,
			&group.MembersCount, &group.IsJoined,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user groups: %w", err)
	}

	return groups, nil
}
