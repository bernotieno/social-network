package models

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	// Import the models package to access GroupPrivacy and PostVisibility
	//  "social-network/backend/pkg/models"
	// "backend/pkg/models"
)

// Comment represents a comment on a post
type Comment struct {
	ID        string    `json:"id"`
	PostID    string    `json:"postId"`
	UserID    string    `json:"userId"`
	Content   string    `json:"content"`
	Image     string    `json:"image,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	// Additional fields for API responses
	Author *User `json:"author,omitempty"`
}

// CommentService handles comment-related operations
type CommentService struct {
	DB *sql.DB
}

// NewCommentService creates a new CommentService
func NewCommentService(db *sql.DB) *CommentService {
	return &CommentService{DB: db}
}

// Create creates a new comment
func (s *CommentService) Create(comment *Comment) error {
	comment.ID = uuid.New().String()
	now := time.Now()
	comment.CreatedAt = now
	comment.UpdatedAt = now

	_, err := s.DB.Exec(`
		INSERT INTO comments (id, post_id, user_id, content, image, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, comment.ID, comment.PostID, comment.UserID, comment.Content, comment.Image, comment.CreatedAt, comment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

// GetByID retrieves a comment by ID
func (s *CommentService) GetByID(id string) (*Comment, error) {
	comment := &Comment{Author: &User{}}
	err := s.DB.QueryRow(`
		SELECT c.id, c.post_id, c.user_id, c.content, c.image, c.created_at, c.updated_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.id = ?
	`, id).Scan(
		&comment.ID, &comment.PostID, &comment.UserID, &comment.Content, &comment.Image, &comment.CreatedAt, &comment.UpdatedAt,
		&comment.Author.ID, &comment.Author.Username, &comment.Author.FullName, &comment.Author.ProfilePicture,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("comment not found")
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	return comment, nil
}

// Update updates a comment
func (s *CommentService) Update(comment *Comment) error {
	comment.UpdatedAt = time.Now()

	_, err := s.DB.Exec(`
		UPDATE comments
		SET content = ?, image = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, comment.Content, comment.Image, comment.UpdatedAt, comment.ID, comment.UserID)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}

// Delete deletes a comment
func (s *CommentService) Delete(id, userID string) error {
	// Check if the user is the comment author or the post owner
	var postOwnerID string
	err := s.DB.QueryRow(`
		SELECT p.user_id
		FROM comments c
		JOIN posts p ON c.post_id = p.id
		WHERE c.id = ?
	`, id).Scan(&postOwnerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("comment not found")
		}
		return fmt.Errorf("failed to check post ownership: %w", err)
	}

	// Delete the comment if the user is the comment author or the post owner
	result, err := s.DB.Exec(`
		DELETE FROM comments
		WHERE id = ? AND (user_id = ? OR ? = ?)
	`, id, userID, userID, postOwnerID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("comment not found or not authorized to delete")
	}

	return nil
}

// GetCommentsByPost retrieves all comments for a post
func (s *CommentService) GetCommentsByPost(postID string, currentUserID string, limit, offset int) ([]*Comment, error) {
	// First, check if the post exists in the regular posts table
	var postUserID string
	var postVisibility PostVisibility
	err := s.DB.QueryRow(
		"SELECT user_id, visibility FROM posts WHERE id = ?",
		postID,
	).Scan(&postUserID, &postVisibility)

	if err != nil {
		if err == sql.ErrNoRows {
			// Post not found in regular posts table, check group_posts table
			var groupPostUserID string
			err = s.DB.QueryRow(
				"SELECT user_id FROM group_posts WHERE id = ?",
				postID,
			).Scan(&groupPostUserID)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, errors.New("post not found for comments")
				}
				log.Printf("GetCommentsByPost: Error checking group post %s: %v", postID, err)
				return nil, fmt.Errorf("failed to check group post: %w", err)
			}

			// For group posts, we don't need to check visibility here as it's handled by the group membership check in the handler
			// Just proceed to get the comments
		} else {
			log.Printf("GetCommentsByPost: Error checking post %s: %v", postID, err)
			return nil, fmt.Errorf("failed to check post: %w", err)
		}
	} else {
		// Regular post found, check if user can view this post's comments
		if postVisibility == PostVisibilityPrivate && postUserID != currentUserID {
			return nil, errors.New("not authorized to view comments on this private post")
		}
		// Add more visibility checks if needed (e.g., followers only)
	}

	rows, err := s.DB.Query(`
		SELECT c.id, c.post_id, c.user_id, c.content, c.image, c.created_at, c.updated_at,
			u.id, u.username, u.full_name, u.profile_picture
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ?
		ORDER BY c.created_at DESC
		LIMIT ? OFFSET ?
	`, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	defer rows.Close()

	var comments []*Comment
	for rows.Next() {
		comment := &Comment{Author: &User{}}
		err := rows.Scan(
			&comment.ID, &comment.PostID, &comment.UserID, &comment.Content, &comment.Image, &comment.CreatedAt, &comment.UpdatedAt,
			&comment.Author.ID, &comment.Author.Username, &comment.Author.FullName, &comment.Author.ProfilePicture,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comments: %w", err)
	}

	return comments, nil
}

// GetCommentCount returns the number of comments for a post
func (s *CommentService) GetCommentCount(postID string) (int, error) {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM comments WHERE post_id = ?", postID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get comment count: %w", err)
	}

	return count, nil
}
