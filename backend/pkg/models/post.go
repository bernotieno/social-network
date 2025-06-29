package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	// Import the models package to access GroupPrivacy
	// _ "social-network/backend/pkg/models"
)

// PostVisibility represents the visibility level of a post
type PostVisibility string

const (
	PostVisibilityPublic    PostVisibility = "public"
	PostVisibilityFollowers PostVisibility = "followers"
	PostVisibilityPrivate   PostVisibility = "private"
	PostVisibilityCustom    PostVisibility = "custom"
)

// Post represents a user post
type Post struct {
	ID         string         `json:"id"`
	UserID     string         `json:"userId"`
	GroupID    sql.NullString `json:"groupId,omitempty"` // New field for group posts
	Content    string         `json:"content"`
	Image      string         `json:"image,omitempty"`
	Visibility PostVisibility `json:"visibility"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	// Additional fields for API responses
	User          *User `json:"author,omitempty"`
	LikesCount    int   `json:"likesCount,omitempty"`
	CommentsCount int   `json:"commentsCount,omitempty"`
	IsLiked       bool  `json:"isLikedByCurrentUser,omitempty"`
}

// PostService handles post-related operations
type PostService struct {
	DB *sql.DB
}

// NewPostService creates a new PostService
func NewPostService(db *sql.DB) *PostService {
	return &PostService{DB: db}
}

// Create creates a new post
func (s *PostService) Create(post *Post) error {
	post.ID = uuid.New().String()
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO posts (id, user_id, content, image, visibility, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, post.ID, post.UserID, post.Content, post.Image, post.Visibility, post.CreatedAt, post.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	return nil
}

// GetByID retrieves a post by ID
func (s *PostService) GetByID(id string, currentUserID string) (*Post, error) {
	post := &Post{User: &User{}}
	var image sql.NullString
	var profilePicture sql.NullString
	var groupID sql.NullString
	err := s.DB.QueryRow(`
		SELECT p.id, p.user_id, p.group_id, p.content, p.image, p.visibility, p.created_at, p.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			(SELECT COUNT(*) FROM likes WHERE post_id = p.id) as likes_count,
			(SELECT COUNT(*) FROM comments WHERE post_id = p.id) as comments_count,
			(SELECT COUNT(*) FROM likes WHERE post_id = p.id AND user_id = ?) as is_liked
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = ?
	`, currentUserID, id).Scan(
		&post.ID, &post.UserID, &groupID, &post.Content, &image, &post.Visibility, &post.CreatedAt, &post.UpdatedAt,
		&post.User.ID, &post.User.Username, &post.User.FullName, &profilePicture,
		&post.LikesCount, &post.CommentsCount, &post.IsLiked,
	)

	if groupID.Valid {
		post.GroupID = groupID
	}

	// Handle nullable fields
	if image.Valid {
		post.Image = image.String
	}
	if profilePicture.Valid {
		post.User.ProfilePicture = profilePicture.String
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("post not found")
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	// Check if the current user can view this post
	// Regular posts don't have group associations, so we only check post visibility
	if post.Visibility != PostVisibilityPublic && post.UserID != currentUserID {
		// For followers-only posts, check if the current user is a follower
		if post.Visibility == PostVisibilityFollowers {
			var isFollowing bool
			err := s.DB.QueryRow(`
				SELECT COUNT(*) > 0
				FROM follows
				WHERE follower_id = ? AND following_id = ? AND status = 'accepted'
			`, currentUserID, post.UserID).Scan(&isFollowing)
			if err != nil {
				return nil, fmt.Errorf("failed to check follow status: %w", err)
			}

			if !isFollowing {
				return nil, errors.New("not authorized to view this post")
			}
		} else if post.Visibility == PostVisibilityCustom {
			// For custom visibility posts, check if the user is in the viewers list
			var canView bool
			err := s.DB.QueryRow(`
				SELECT COUNT(*) > 0
				FROM post_viewers
				WHERE post_id = ? AND user_id = ?
			`, post.ID, currentUserID).Scan(&canView)
			if err != nil {
				return nil, fmt.Errorf("failed to check custom viewer permission: %w", err)
			}

			if !canView {
				return nil, errors.New("not authorized to view this post")
			}
		} else {
			// Private posts can only be viewed by the owner
			return nil, errors.New("not authorized to view this post")
		}
	}

	return post, nil
}

// Update updates a post
func (s *PostService) Update(post *Post) error {
	post.UpdatedAt = time.Now()

	_, err := s.DB.Exec(`
		UPDATE posts
		SET content = ?, image = ?, visibility = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, post.Content, post.Image, post.Visibility, post.UpdatedAt, post.ID, post.UserID)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	return nil
}

// Delete deletes a post
func (s *PostService) Delete(id, userID string) error {
	result, err := s.DB.Exec(`
		DELETE FROM posts
		WHERE id = ? AND user_id = ?
	`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("post not found or not authorized to delete")
	}

	return nil
}

// GetUserPosts retrieves posts by a user
func (s *PostService) GetUserPosts(userID, currentUserID string, limit, offset int) ([]*Post, error) {
	// First, check if the user has a private profile
	var isPrivate bool
	err := s.DB.QueryRow("SELECT is_private FROM users WHERE id = ?", userID).Scan(&isPrivate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to check user privacy: %w", err)
	}

	// Handle empty currentUserID (unauthenticated users)
	if currentUserID == "" {
		currentUserID = "00000000-0000-0000-0000-000000000000" // Use a dummy UUID that won't match any real user
	}

	// If user is viewing their own posts or the profile is public, proceed normally
	if userID == currentUserID || !isPrivate {
		// User can see all their own posts or public profile posts
		rows, err := s.DB.Query(`
			SELECT p.id, p.user_id, p.content, p.image, p.visibility, p.created_at, p.updated_at,
				u.id, u.username, u.full_name, u.profile_picture,
				(SELECT COUNT(*) FROM likes WHERE post_id = p.id) as likes_count,
				(SELECT COUNT(*) FROM comments WHERE post_id = p.id) as comments_count,
				COALESCE((SELECT COUNT(*) FROM likes WHERE post_id = p.id AND user_id = ?), 0) as is_liked
			FROM posts p
			JOIN users u ON p.user_id = u.id
			WHERE p.user_id = ?
			ORDER BY p.created_at DESC
			LIMIT ? OFFSET ?
		`, currentUserID, userID, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to get user posts: %w", err)
		}
		defer rows.Close()

		return s.scanPosts(rows)
	} else {
		// For private profiles, check if the current user is a follower
		var isFollowing bool
		err := s.DB.QueryRow(`
			SELECT COUNT(*) > 0
			FROM follows
			WHERE follower_id = ? AND following_id = ? AND status = 'accepted'
		`, currentUserID, userID).Scan(&isFollowing)
		if err != nil {
			return nil, fmt.Errorf("failed to check follow status: %w", err)
		}

		if !isFollowing {
			// Not a follower, return empty list or error based on your preference
			return []*Post{}, nil // Or return nil, errors.New("not authorized to view posts")
		}

		// Is a follower, can see posts
		rows, err := s.DB.Query(`
			SELECT p.id, p.user_id, p.content, p.image, p.visibility, p.created_at, p.updated_at,
				u.id, u.username, u.full_name, u.profile_picture,
				(SELECT COUNT(*) FROM likes WHERE post_id = p.id) as likes_count,
				(SELECT COUNT(*) FROM comments WHERE post_id = p.id) as comments_count,
				COALESCE((SELECT COUNT(*) FROM likes WHERE post_id = p.id AND user_id = ?), 0) as is_liked
			FROM posts p
			JOIN users u ON p.user_id = u.id
			WHERE p.user_id = ?
			ORDER BY p.created_at DESC
			LIMIT ? OFFSET ?
		`, currentUserID, userID, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to get user posts: %w", err)
		}
		defer rows.Close()

		return s.scanPosts(rows)
	}
}

// Helper method to scan posts from rows
func (s *PostService) scanPosts(rows *sql.Rows) ([]*Post, error) {
	var posts []*Post
	for rows.Next() {
		post := &Post{User: &User{}}
		var image sql.NullString
		var profilePicture sql.NullString
		err := rows.Scan(
			&post.ID, &post.UserID, &post.Content, &image, &post.Visibility, &post.CreatedAt, &post.UpdatedAt,
			&post.User.ID, &post.User.Username, &post.User.FullName, &profilePicture,
			&post.LikesCount, &post.CommentsCount, &post.IsLiked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		// Handle nullable fields
		if image.Valid {
			post.Image = image.String
		}
		if profilePicture.Valid {
			post.User.ProfilePicture = profilePicture.String
		}

		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating posts: %w", err)
	}

	return posts, nil
}

// GetFeed retrieves posts for a user's feed
func (s *PostService) GetFeed(userID string, limit, offset int) ([]*Post, error) {
	rows, err := s.DB.Query(`
		SELECT p.id, p.user_id, p.content, p.image, p.visibility, p.created_at, p.updated_at,
			u.id, u.username, u.full_name, u.profile_picture,
			(SELECT COUNT(*) FROM likes WHERE post_id = p.id) as likes_count,
			(SELECT COUNT(*) FROM comments WHERE post_id = p.id) as comments_count,
			(SELECT COUNT(*) FROM likes WHERE post_id = p.id AND user_id = ?) as is_liked
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE
			-- Include user's own posts (all visibility levels)
			p.user_id = ?
			-- Include public posts from users the user is following
			OR (p.visibility = ? AND p.user_id IN (
				SELECT following_id FROM follows WHERE follower_id = ? AND status = 'accepted'
			))
			-- Include followers-only posts from users the user is following
			OR (p.visibility = ? AND p.user_id IN (
				SELECT following_id FROM follows WHERE follower_id = ? AND status = 'accepted'
			))
			-- Include public posts from users with public profiles (not following)
			OR (p.visibility = ? AND p.user_id IN (
				SELECT id FROM users WHERE is_private = FALSE
			) AND p.user_id NOT IN (
				SELECT following_id FROM follows WHERE follower_id = ? AND status = 'accepted'
			))
			-- Include custom visibility posts where the user is in the viewers list
			OR (p.visibility = ? AND p.id IN (
				SELECT post_id FROM post_viewers WHERE user_id = ?
			))
			-- Note: Private posts are only visible to the owner (handled by first condition)
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`, userID, userID, PostVisibilityPublic, userID, PostVisibilityFollowers, userID, PostVisibilityPublic, userID, PostVisibilityCustom, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		post := &Post{User: &User{}}
		var image sql.NullString
		var profilePicture sql.NullString
		err := rows.Scan(
			&post.ID, &post.UserID, &post.Content, &image, &post.Visibility, &post.CreatedAt, &post.UpdatedAt,
			&post.User.ID, &post.User.Username, &post.User.FullName, &profilePicture,
			&post.LikesCount, &post.CommentsCount, &post.IsLiked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		// Handle nullable fields
		if image.Valid {
			post.Image = image.String
		}
		if profilePicture.Valid {
			post.User.ProfilePicture = profilePicture.String
		}

		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feed posts: %w", err)
	}

	return posts, nil
}
