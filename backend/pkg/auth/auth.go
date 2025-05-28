package auth

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/bernaotieno/social-network/backend/pkg/models"
	"github.com/gorilla/sessions"
)

// Session cookie name
const (
	SessionCookieName = "social_network_session"
	SessionDuration   = 7 * 24 * time.Hour // 7 days
)

// Store is the session store
var Store *sessions.CookieStore

// Initialize initializes the auth package
func Initialize(secret []byte) {
	Store = sessions.NewCookieStore(secret)
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
}

// CreateSession creates a new session for a user
func CreateSession(ctx context.Context, db *sql.DB, userID string, w http.ResponseWriter, r *http.Request) (string, error) {
	// Create a new session in the database
	sessionService := models.NewSessionService(db)
	session, err := sessionService.Create(userID, SessionDuration)
	if err != nil {
		return "", err
	}

	// Create a new cookie session
	cookieSession, err := Store.Get(r, SessionCookieName)
	if err != nil {
		return "", err
	}
	log.Println("cookie session", cookieSession)
	// Set session ID in cookie
	cookieSession.Values["session_id"] = session.ID
	if err := cookieSession.Save(r, w); err != nil {
		return "", err
	}

	return session.ID, nil
}

// GetSessionCookie gets the session ID from the cookie
func GetSessionCookie(r *http.Request) (string, error) {
	session, err := Store.Get(r, SessionCookieName)
	if err != nil {
		return "", err
	}

	// Get session ID from cookie
	sessionID, ok := session.Values["session_id"].(string)
	if !ok {
		return "", errors.New("session ID not found in cookie")
	}

	return sessionID, nil
}

// ValidateSession validates a session and returns the user ID
func ValidateSession(ctx context.Context, db *sql.DB, sessionID string) (string, error) {
	// Validate session in the database
	sessionService := models.NewSessionService(db)
	session, err := sessionService.GetByID(sessionID)
	if err != nil {
		return "", err
	}

	return session.UserID, nil
}

// ClearSession clears the session cookie and removes the session from the database
func ClearSession(ctx context.Context, db *sql.DB, w http.ResponseWriter, r *http.Request) error {
	// Get session ID from cookie
	sessionID, err := GetSessionCookie(r)
	if err != nil {
		return err
	}

	// Delete session from database
	sessionService := models.NewSessionService(db)
	if err := sessionService.Delete(sessionID); err != nil {
		return err
	}

	// Clear cookie session
	session, err := Store.Get(r, SessionCookieName)
	if err != nil {
		return err
	}

	// Delete the cookie by setting MaxAge to -1
	session.Options.MaxAge = -1
	return session.Save(r, w)
}
