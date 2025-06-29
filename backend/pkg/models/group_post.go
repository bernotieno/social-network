package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GroupPost represents a post in a group
type GroupPost struct {
	ID        string    `json:"id"`
	GroupID   string    `json:"groupId"`
	UserID    string    `json:"userId"`
	Content   string    `json:"content"`
	Image     string    `json:"image,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	// Additional fields for API responses
	User          *User  `json:"author,omitempty"`
	Group         *Group `json:"group,omitempty"`
	LikesCount    int    `json:"likesCount,omitempty"`
	CommentsCount int    `json:"commentsCount,omitempty"`
	IsLiked       bool   `json:"isLikedByCurrentUser,omitempty"`
}

// GroupPostService handles group post-related operations
type GroupPostService struct {
	DB *sql.DB
}

// NewGroupPostService creates a new GroupPostService
func NewGroupPostService(db *sql.DB) *GroupPostService {
	return &GroupPostService{DB: db}
}

// Create creates a new group post
func (s *GroupPostService) Create(post *GroupPost) error {
	post.ID = uuid.New().String()
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO group_posts (id, group_id, user_id, content, image, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, post.ID, post.GroupID, post.UserID, post.Content, post.Image, post.CreatedAt, post.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create group post: %w", err)
	}

	return nil
}

// GetByID retrieves a group post by ID
func (s *GroupPostService) GetByID(id string, currentUserID string) (*GroupPost, error) {
	post := &GroupPost{User: &User{}, Group: &Group{}}
	var isLikedCount int
	err := s.DB.QueryRow(`
		SELECT gp.id, gp.group_id, gp.user_id, gp.content, gp.image, gp.created_at, gp.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			g.id, g.name, g.privacy,
			(SELECT COUNT(*) FROM likes WHERE post_id = gp.id) as likes_count,
			(SELECT COUNT(*) FROM comments WHERE post_id = gp.id) as comments_count,
			(SELECT COUNT(*) FROM likes WHERE post_id = gp.id AND user_id = ?) as is_liked
		FROM group_posts gp
		JOIN users u ON gp.user_id = u.id
		JOIN groups g ON gp.group_id = g.id
		WHERE gp.id = ?
	`, currentUserID, id).Scan(
		&post.ID, &post.GroupID, &post.UserID, &post.Content, &post.Image, &post.CreatedAt, &post.UpdatedAt,
		&post.User.ID, &post.User.Username, &post.User.FullName, &post.User.ProfilePicture,
		&post.Group.ID, &post.Group.Name, &post.Group.Privacy,
		&post.LikesCount, &post.CommentsCount, &isLikedCount,
	)

	if err == nil {
		post.IsLiked = isLikedCount > 0
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("group post not found")
		}
		return nil, fmt.Errorf("failed to get group post: %w", err)
	}

	// Check if the current user can view this post
	if post.Group.Privacy == GroupPrivacyPrivate {
		// Check if the current user is a member of the group
		var isMember bool
		err := s.DB.QueryRow(`
			SELECT COUNT(*) > 0
			FROM group_members
			WHERE group_id = ? AND user_id = ? AND status = 'accepted'
		`, post.GroupID, currentUserID).Scan(&isMember)
		if err != nil {
			return nil, fmt.Errorf("failed to check group membership: %w", err)
		}

		if !isMember {
			return nil, errors.New("not authorized to view this post")
		}
	}

	return post, nil
}

// Update updates a group post
func (s *GroupPostService) Update(post *GroupPost) error {
	post.UpdatedAt = time.Now()

	_, err := s.DB.Exec(`
		UPDATE group_posts
		SET content = ?, image = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, post.Content, post.Image, post.UpdatedAt, post.ID, post.UserID)
	if err != nil {
		return fmt.Errorf("failed to update group post: %w", err)
	}

	return nil
}

// Delete deletes a group post
func (s *GroupPostService) Delete(id, userID string) error {
	// Check if user is the post author or a group admin
	var groupID string
	var postAuthorID string
	err := s.DB.QueryRow(`
		SELECT group_id, user_id
		FROM group_posts
		WHERE id = ?
	`, id).Scan(&groupID, &postAuthorID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("group post not found")
		}
		return fmt.Errorf("failed to check post ownership: %w", err)
	}

	// Check if user is the post author
	if postAuthorID == userID {
		// User is the author, allow deletion
		_, err = s.DB.Exec("DELETE FROM group_posts WHERE id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to delete group post: %w", err)
		}
		return nil
	}

	// If not the author, check group permissions
	var memberRole GroupMemberRole
	var groupCreatorID string

	// Get the role of the user in the group and the group's creator ID
	err = s.DB.QueryRow(`
		SELECT gm.role, g.creator_id
		FROM group_members gm
		JOIN groups g ON gm.group_id = g.id
		WHERE gm.group_id = ? AND gm.user_id = ? AND gm.status = 'accepted'
	`, groupID, userID).Scan(&memberRole, &groupCreatorID)

	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("user is not an active member of this group")
		}
		return fmt.Errorf("failed to get group member role: %w", err)
	}

	// Group creator can delete any post
	if userID == groupCreatorID {
		_, err = s.DB.Exec("DELETE FROM group_posts WHERE id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to delete group post: %w", err)
		}
		return nil
	}

	// Admin can delete any post except the creator's post
	if memberRole == GroupMemberRoleAdmin {
		// Get the author's role to ensure admin cannot delete creator's post
		var postAuthorRole GroupMemberRole
		err = s.DB.QueryRow(`
			SELECT role
			FROM group_members
			WHERE group_id = ? AND user_id = ? AND status = 'accepted'
		`, groupID, postAuthorID).Scan(&postAuthorRole)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("failed to get post author's role: %w", err)
		} else if err == sql.ErrNoRows {
			// If the post author is not a group member, treat them as a regular member for permission checks
			postAuthorRole = GroupMemberRoleMember
		}

		// If the post author is the creator, admin cannot delete it
		if postAuthorID == groupCreatorID || postAuthorRole == GroupMemberRoleCreator {
			return errors.New("admin cannot delete the group creator's post")
		}

		// Admin can delete the post if it's not the creator's
		_, err = s.DB.Exec("DELETE FROM group_posts WHERE id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to delete group post: %w", err)
		}
		return nil
	}

	// If none of the above conditions are met, the user is not authorized to delete the post
	return errors.New("not authorized to delete this post")
}

// GetByGroup retrieves posts for a group
func (s *GroupPostService) GetByGroup(groupID, currentUserID string, limit, offset int) ([]*GroupPost, error) {
	// Check if the current user can view posts in this group
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
			return nil, errors.New("not authorized to view posts in this group")
		}
	}

	// Get posts
	rows, err := s.DB.Query(`
		SELECT gp.id, gp.group_id, gp.user_id, gp.content, gp.image, gp.created_at, gp.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			(SELECT COUNT(*) FROM likes WHERE post_id = gp.id) as likes_count,
			(SELECT COUNT(*) FROM comments WHERE post_id = gp.id) as comments_count,
			(SELECT COUNT(*) FROM likes WHERE post_id = gp.id AND user_id = ?) as is_liked
		FROM group_posts gp
		JOIN users u ON gp.user_id = u.id
		WHERE gp.group_id = ?
		ORDER BY gp.created_at DESC
		LIMIT ? OFFSET ?
	`, currentUserID, groupID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get group posts: %w", err)
	}
	defer rows.Close()

	var posts []*GroupPost
	for rows.Next() {
		post := &GroupPost{User: &User{}}
		var isLikedCount int
		err := rows.Scan(
			&post.ID, &post.GroupID, &post.UserID, &post.Content, &post.Image, &post.CreatedAt, &post.UpdatedAt,
			&post.User.ID, &post.User.Username, &post.User.FullName, &post.User.ProfilePicture,
			&post.LikesCount, &post.CommentsCount, &isLikedCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group post: %w", err)
		}
		post.IsLiked = isLikedCount > 0
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating group posts: %w", err)
	}

	return posts, nil
}
