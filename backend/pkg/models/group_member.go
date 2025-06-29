package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GroupMemberRole represents the role of a group member

// GroupMember represents a member of a group
type GroupMember struct {
	ID        string            `json:"id"`
	GroupID   string            `json:"groupId"`
	UserID    string            `json:"userId"`
	Role      GroupMemberRole   `json:"role"`
	Status    GroupMemberStatus `json:"status"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
	// Additional fields for API responses
	User  *User  `json:"user,omitempty"`
	Group *Group `json:"group,omitempty"`
}

// GroupMemberService handles group member-related operations
type GroupMemberService struct {
	DB *sql.DB
}

// NewGroupMemberService creates a new GroupMemberService
func NewGroupMemberService(db *sql.DB) *GroupMemberService {
	return &GroupMemberService{DB: db}
}

// Create creates a new group member
func (s *GroupMemberService) Create(member *GroupMember) error {
	member.ID = uuid.New().String()
	now := time.Now()
	member.CreatedAt = now
	member.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO group_members (id, group_id, user_id, role, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, member.ID, member.GroupID, member.UserID, member.Role, member.Status, member.CreatedAt, member.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create group member: %w", err)
	}

	return nil
}

// GetByID retrieves a group member by ID
func (s *GroupMemberService) GetByID(id string) (*GroupMember, error) {
	member := &GroupMember{User: &User{}, Group: &Group{}}
	err := s.DB.QueryRow(`
		SELECT gm.id, gm.group_id, gm.user_id, gm.role, gm.status, gm.created_at, gm.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			g.id, g.name, g.privacy
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		JOIN groups g ON gm.group_id = g.id
		WHERE gm.id = ?
	`, id).Scan(
		&member.ID, &member.GroupID, &member.UserID, &member.Role, &member.Status, &member.CreatedAt, &member.UpdatedAt,
		&member.User.ID, &member.User.Username, &member.User.FullName, &member.User.ProfilePicture,
		&member.Group.ID, &member.Group.Name, &member.Group.Privacy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("group member not found")
		}
		return nil, fmt.Errorf("failed to get group member: %w", err)
	}

	return member, nil
}

// GetByGroupAndUser retrieves a group member by group ID and user ID
func (s *GroupMemberService) GetByGroupAndUser(groupID, userID string) (*GroupMember, error) {
	member := &GroupMember{}
	err := s.DB.QueryRow(`
		SELECT id, group_id, user_id, role, status, created_at, updated_at
		FROM group_members
		WHERE group_id = ? AND user_id = ?
	`, groupID, userID).Scan(
		&member.ID, &member.GroupID, &member.UserID, &member.Role, &member.Status, &member.CreatedAt, &member.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("group member not found")
		}
		return nil, fmt.Errorf("failed to get group member: %w", err)
	}

	return member, nil
}

// UpdateStatus updates the status of a group member
func (s *GroupMemberService) UpdateStatus(id string, status GroupMemberStatus) error {
	_, err := s.DB.Exec(`
		UPDATE group_members
		SET status = ?, updated_at = ?
		WHERE id = ?
	`, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update group member status: %w", err)
	}

	return nil
}

// UpdateRole updates the role of a group member
func (s *GroupMemberService) UpdateRole(id string, role GroupMemberRole) error {
	_, err := s.DB.Exec(`
		UPDATE group_members
		SET role = ?, updated_at = ?
		WHERE id = ?
	`, role, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update group member role: %w", err)
	}

	return nil
}

// PromoteToAdmin promotes a group member to admin
func (s *GroupMemberService) PromoteToAdmin(groupID, memberID, callerID string) error {
	// Get the group to check creator
	var creatorID string
	err := s.DB.QueryRow("SELECT creator_id FROM groups WHERE id = ?", groupID).Scan(&creatorID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("group not found")
		}
		return fmt.Errorf("failed to get group creator: %w", err)
	}

	// Only the group creator can promote members
	if creatorID != callerID {
		return errors.New("only the group creator can promote members")
	}

	// Prevent promoting the group creator
	if memberID == creatorID {
		return errors.New("cannot promote the group creator")
	}

	// Update the member's role to admin
	_, err = s.DB.Exec(`
		UPDATE group_members
		SET role = ?, updated_at = ?
		WHERE group_id = ? AND user_id = ?
	`, GroupMemberRoleAdmin, time.Now(), groupID, memberID)
	if err != nil {
		return fmt.Errorf("failed to promote member to admin: %w", err)
	}

	return nil
}

// DemoteFromAdmin demotes a group admin to a regular member
func (s *GroupMemberService) DemoteFromAdmin(groupID, memberID, callerID string) error {
	// Get the group to check creator
	var creatorID string
	err := s.DB.QueryRow("SELECT creator_id FROM groups WHERE id = ?", groupID).Scan(&creatorID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("group not found")
		}
		return fmt.Errorf("failed to get group creator: %w", err)
	}

	// Only the group creator can demote admins
	if creatorID != callerID {
		return errors.New("only the group creator can demote admins")
	}

	// Prevent demoting the group creator
	if memberID == creatorID {
		return errors.New("cannot demote the group creator")
	}

	// Update the member's role to member
	_, err = s.DB.Exec(`
		UPDATE group_members
		SET role = ?, updated_at = ?
		WHERE group_id = ? AND user_id = ?
	`, GroupMemberRoleMember, time.Now(), groupID, memberID)
	if err != nil {
		return fmt.Errorf("failed to demote admin: %w", err)
	}

	return nil
}

// RemoveMember removes a member from a group based on roles
func (s *GroupMemberService) RemoveMember(groupID, memberID, callerID string) error {
	// Get the group to check creator
	var creatorID string
	err := s.DB.QueryRow("SELECT creator_id FROM groups WHERE id = ?", groupID).Scan(&creatorID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("group not found")
		}
		return fmt.Errorf("failed to get group creator: %w", err)
	}

	// Get the member to be removed
	member, err := s.GetByGroupAndUser(groupID, memberID)
	if err != nil {
		return fmt.Errorf("member not found in group: %w", err)
	}

	// Get the caller's role
	callerMember, err := s.GetByGroupAndUser(groupID, callerID)
	// If caller is not a member, they can only remove if they are the creator
	if err != nil && creatorID != callerID {
		return errors.New("not authorized to remove members from this group")
	}

	// Logic for removal:
	// 1. Group creator can remove any member (including other admins).
	// 2. Other admins can only remove regular members.

	if creatorID == callerID {
		// Creator can remove anyone
		// Proceed to delete
	} else if callerMember != nil && callerMember.Role == GroupMemberRoleAdmin {
		// Admin can only remove regular members
		if member.Role == GroupMemberRoleAdmin || member.Role == GroupMemberRoleMember {
			// Admins cannot remove other admins or themselves if they are admin
			if memberID == callerID {
				return errors.New("admins cannot remove themselves")
			}
			// Admins can remove regular members
			if member.Role == GroupMemberRoleMember {
				// Proceed to delete
			} else {
				return errors.New("admins can only remove regular members")
			}
		} else {
			return errors.New("admins can only remove regular members")
		}
	} else {
		return errors.New("not authorized to remove members from this group")
	}

	// Delete the member
	_, err = s.DB.Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = ?", groupID, memberID)
	if err != nil {
		return fmt.Errorf("failed to remove group member: %w", err)
	}

	return nil
}

// Delete deletes a group member
// This function is now a wrapper around RemoveMember to ensure role-based logic is applied.
// The `callerID` for this function would typically be the user initiating the delete action.
// For simplicity, if this function is called directly without a specific caller context,
// it might imply an internal system operation or a creator-initiated removal.
// However, for explicit role-based removal, `RemoveMember` should be used.
func (s *GroupMemberService) Delete(groupID, userID string) error {
	// In a real application, you might need to pass the actual callerID here.
	// For now, assuming the caller is the group creator or an authorized system process
	// if this function is called directly.
	// A more robust solution would involve refactoring handlers to call RemoveMember directly
	// with the authenticated user's ID.
	return s.RemoveMember(groupID, userID, "") // Placeholder for callerID, needs proper context
}

// GetMembers retrieves members of a group
func (s *GroupMemberService) GetMembers(groupID string, limit, offset int) ([]*GroupMember, error) {
	rows, err := s.DB.Query(`
		SELECT gm.id, gm.group_id, gm.user_id, gm.role, gm.status, gm.created_at, gm.updated_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = ? AND gm.status = 'accepted'
		ORDER BY gm.role = 'admin' DESC, gm.created_at ASC
		LIMIT ? OFFSET ?
	`, groupID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer rows.Close()

	var members []*GroupMember
	for rows.Next() {
		member := &GroupMember{User: &User{}}
		err := rows.Scan(
			&member.ID, &member.GroupID, &member.UserID, &member.Role, &member.Status, &member.CreatedAt, &member.UpdatedAt,
			&member.User.ID, &member.User.Username, &member.User.FullName, &member.User.ProfilePicture,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group member: %w", err)
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating group members: %w", err)
	}

	return members, nil
}

// GetPendingRequests retrieves pending join requests for a group
func (s *GroupMemberService) GetPendingRequests(groupID string) ([]*GroupMember, error) {
	rows, err := s.DB.Query(`
		SELECT gm.id, gm.group_id, gm.user_id, gm.role, gm.status, gm.created_at, gm.updated_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = ? AND gm.status = 'pending'
		ORDER BY gm.created_at ASC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending requests: %w", err)
	}
	defer rows.Close()

	var members []*GroupMember
	for rows.Next() {
		member := &GroupMember{User: &User{}}
		err := rows.Scan(
			&member.ID, &member.GroupID, &member.UserID, &member.Role, &member.Status, &member.CreatedAt, &member.UpdatedAt,
			&member.User.ID, &member.User.Username, &member.User.FullName, &member.User.ProfilePicture,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group member: %w", err)
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending requests: %w", err)
	}

	return members, nil
}

// GetInvitedUsers retrieves users who have been invited to a group but haven't responded yet
func (s *GroupMemberService) GetInvitedUsers(groupID string) ([]*GroupMember, error) {
	rows, err := s.DB.Query(`
		SELECT gm.id, gm.group_id, gm.user_id, gm.role, gm.status, gm.created_at, gm.updated_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = ? AND gm.status = 'invited'
		ORDER BY gm.created_at ASC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invited users: %w", err)
	}
	defer rows.Close()

	var members []*GroupMember
	for rows.Next() {
		member := &GroupMember{User: &User{}}
		err := rows.Scan(
			&member.ID, &member.GroupID, &member.UserID, &member.Role, &member.Status, &member.CreatedAt, &member.UpdatedAt,
			&member.User.ID, &member.User.Username, &member.User.FullName, &member.User.ProfilePicture,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invited user: %w", err)
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating invited users: %w", err)
	}

	return members, nil
}

// IsGroupAdmin checks if a user is an admin of a group
func (s *GroupMemberService) IsGroupAdmin(groupID, userID string) (bool, error) {
	var count int
	err := s.DB.QueryRow(`
		SELECT COUNT(*)
		FROM (
			SELECT 1 FROM groups WHERE id = ? AND creator_id = ?
			UNION
			SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ? AND role = 'admin' AND status = 'accepted'
		) AS admin_check
	`, groupID, userID, groupID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if user is admin: %w", err)
	}

	return count > 0, nil
}

// IsGroupMember checks if a user is a member of a group
func (s *GroupMemberService) IsGroupMember(groupID, userID string) (bool, error) {
	var count int
	err := s.DB.QueryRow(`
		SELECT COUNT(*)
		FROM (
			SELECT 1 FROM groups WHERE id = ? AND creator_id = ?
			UNION
			SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ? AND status = 'accepted'
		) AS membership
	`, groupID, userID, groupID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if user is member: %w", err)
	}

	return count > 0, nil
}

// Update updates a group member
func (s *GroupMemberService) Update(member *GroupMember) error {
	member.UpdatedAt = time.Now()

	_, err := s.DB.Exec(`
		UPDATE group_members
		SET role = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, member.Role, member.Status, member.UpdatedAt, member.ID)
	if err != nil {
		return fmt.Errorf("failed to update group member: %w", err)
	}

	return nil
}

// GetGroupAdmins retrieves all admins of a group (including creator)
func (s *GroupMemberService) GetGroupAdmins(groupID string) ([]*GroupMember, error) {
	// First get all admin members
	rows, err := s.DB.Query(`
		SELECT gm.id, gm.group_id, gm.user_id, gm.role, gm.status, gm.created_at, gm.updated_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = ? AND gm.role = 'admin' AND gm.status = 'accepted'
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group admins: %w", err)
	}
	defer rows.Close()

	var admins []*GroupMember
	adminUserIDs := make(map[string]bool) // Track unique user IDs

	// Add all admin members
	for rows.Next() {
		admin := &GroupMember{User: &User{}}
		err := rows.Scan(
			&admin.ID, &admin.GroupID, &admin.UserID, &admin.Role, &admin.Status, &admin.CreatedAt, &admin.UpdatedAt,
			&admin.User.ID, &admin.User.Username, &admin.User.FullName, &admin.User.ProfilePicture,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group admin: %w", err)
		}

		admins = append(admins, admin)
		adminUserIDs[admin.UserID] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating group admins: %w", err)
	}

	// Now get the group creator if they're not already included as an admin
	var creatorID string
	err = s.DB.QueryRow(`
		SELECT g.creator_id
		FROM groups g
		WHERE g.id = ?
	`, groupID).Scan(&creatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group creator: %w", err)
	}

	// If creator is not already in the admins list, add them
	if !adminUserIDs[creatorID] {
		var creator GroupMember
		creator.User = &User{}
		err = s.DB.QueryRow(`
			SELECT u.id, u.username, u.full_name, u.profile_picture
			FROM users u
			WHERE u.id = ?
		`, creatorID).Scan(
			&creator.User.ID, &creator.User.Username, &creator.User.FullName, &creator.User.ProfilePicture,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get creator user info: %w", err)
		}

		creator.GroupID = groupID
		creator.UserID = creatorID
		creator.Role = GroupMemberRoleAdmin
		creator.Status = GroupMemberStatusAccepted
		admins = append(admins, &creator)
	}

	return admins, nil
}
