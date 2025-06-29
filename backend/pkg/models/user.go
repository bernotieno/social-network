package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserRole defines the role of a user.
type UserRole string

const (
	UserRoleAdmin     UserRole = "admin"
	UserRoleMember    UserRole = "member"
	UserRoleModerator UserRole = "moderator"
)

// User represents a user in the system
type User struct {
	ID             string     `json:"id"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	Password       string     `json:"-"` // Never expose password in JSON
	FullName       string     `json:"fullName"`
	FirstName      string     `json:"firstName"`
	LastName       string     `json:"lastName"`
	DateOfBirth    *time.Time `json:"dateOfBirth,omitempty"`
	Bio            string     `json:"bio,omitempty"`
	ProfilePicture string     `json:"profilePicture,omitempty"`
	CoverPhoto     string     `json:"coverPhoto,omitempty"`
	IsPrivate      bool       `json:"isPrivate"`
	Role           string     `json:"role"` // Added Role field
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// UserService handles user-related operations
type UserService struct {
	DB *sql.DB
}

// NewUserService creates a new UserService
func NewUserService(db *sql.DB) *UserService {
	return &UserService{DB: db}
}

// Create creates a new user
func (s *UserService) Create(user *User) error {
	// Generate UUID for the user
	user.ID = uuid.New().String()

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Set timestamps
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Insert user into database
	_, err = s.DB.Exec(`
		INSERT INTO users (id, username, email, password, full_name, first_name, last_name, date_of_birth, bio, profile_picture, cover_photo, is_private, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, user.ID, user.Username, user.Email, string(hashedPassword), user.FullName, user.FirstName, user.LastName, user.DateOfBirth, user.Bio, user.ProfilePicture, user.CoverPhoto, user.IsPrivate, user.Role, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(id string) (*User, error) {
	user := &User{}
	err := s.DB.QueryRow(`
		SELECT id, username, email, password, full_name, first_name, last_name, date_of_birth, bio, profile_picture, cover_photo, is_private, role, created_at, updated_at
	FROM users
	WHERE id = ?
	`, id).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.FullName, &user.FirstName, &user.LastName, &user.DateOfBirth, &user.Bio, &user.ProfilePicture, &user.CoverPhoto, &user.IsPrivate, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (s *UserService) GetByEmail(email string) (*User, error) {
	user := &User{}
	err := s.DB.QueryRow(`
		SELECT id, username, email, password, full_name, first_name, last_name, date_of_birth, bio, profile_picture, cover_photo, is_private, role, created_at, updated_at
	FROM users
	WHERE email = ?
	`, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.FullName, &user.FirstName, &user.LastName, &user.DateOfBirth, &user.Bio, &user.ProfilePicture, &user.CoverPhoto, &user.IsPrivate, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByUsername retrieves a user by username
func (s *UserService) GetByUsername(username string) (*User, error) {
	user := &User{}
	err := s.DB.QueryRow(`
		SELECT id, username, email, password, full_name, first_name, last_name, date_of_birth, bio, profile_picture, cover_photo, is_private, role, created_at, updated_at
	FROM users
	WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.FullName, &user.FirstName, &user.LastName, &user.DateOfBirth, &user.Bio, &user.ProfilePicture, &user.CoverPhoto, &user.IsPrivate, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// Update updates a user's profile
func (s *UserService) Update(user *User) error {
	user.UpdatedAt = time.Now()

	_, err := s.DB.Exec(`
		UPDATE users
		SET username = ?, email = ?, full_name = ?, first_name = ?, last_name = ?, date_of_birth = ?, bio = ?, profile_picture = ?, cover_photo = ?, is_private = ?, updated_at = ?
		WHERE id = ?
	`, user.Username, user.Email, user.FullName, user.FirstName, user.LastName, user.DateOfBirth, user.Bio, user.ProfilePicture, user.CoverPhoto, user.IsPrivate, user.UpdatedAt, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdatePassword updates a user's password
func (s *UserService) UpdatePassword(userID, newPassword string) error {
	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update the password in the database
	_, err = s.DB.Exec(`
		UPDATE users
		SET password = ?, updated_at = ?
		WHERE id = ?
	`, string(hashedPassword), time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// CheckPassword checks if the provided password matches the user's password
func (s *UserService) CheckPassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

// GetUsers retrieves users with optional filtering
func (s *UserService) GetUsers(query string, limit, offset int) ([]*User, error) {
	var rows *sql.Rows
	var err error

	if query != "" {
		rows, err = s.DB.Query(`
			SELECT id, username, email, password, full_name, first_name, last_name, date_of_birth, bio, profile_picture, cover_photo, is_private, role, created_at, updated_at
			FROM users
			WHERE username LIKE ? OR full_name LIKE ? OR first_name LIKE ? OR last_name LIKE ?
			LIMIT ? OFFSET ?
		`, "%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%", limit, offset)
	} else {
		rows, err = s.DB.Query(`
			SELECT id, username, email, password, full_name, first_name, last_name, date_of_birth, bio, profile_picture, cover_photo, is_private, role, created_at, updated_at
			FROM users
			LIMIT ? OFFSET ?
		`, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.FullName, &user.FirstName, &user.LastName, &user.DateOfBirth, &user.Bio, &user.ProfilePicture, &user.CoverPhoto, &user.IsPrivate, &user.Role, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// UserExists checks if a user with the given email or username exists
func (s *UserService) UserExists(email, username string) (bool, error) {
	var count int
	err := s.DB.QueryRow(`
		SELECT COUNT(*) FROM users WHERE email = ? OR username = ?
	`, email, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}

	return count > 0, nil
}
