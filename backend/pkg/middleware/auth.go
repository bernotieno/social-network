package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/bernaotieno/social-network/backend/pkg/auth"
	"github.com/bernaotieno/social-network/backend/pkg/utils"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// UserIDKey is the key used to store the user ID in the request context
const UserIDKey contextKey = "userID"

// DBKey is the key used to store the database connection in the request context
const DBKey contextKey = "db"

// AuthMiddleware authenticates the user and adds the user ID to the request context
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get database connection from context
		db, ok := r.Context().Value(DBKey).(*sql.DB)
		if !ok {
			utils.RespondWithError(w, http.StatusInternalServerError, "Database connection not found")
			return
		}

		// Try cookie authentication first
		sessionID, err := auth.GetSessionCookie(r)
		if err == nil {
			// Validate session
			userID, err := auth.ValidateSession(r.Context(), db, sessionID)
			if err == nil {
				// Add user ID to request context
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				next(w, r.WithContext(ctx))
				return
			}
		}

		// If cookie auth fails, try Bearer token authentication
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized: No authentication provided")
			return
		}

		// Check if the header has the correct format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized: Invalid authorization format")
			return
		}

		// Validate token
		userID, err := auth.ValidateSession(r.Context(), db, parts[1])
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized: "+err.Error())
			return
		}

		// Add user ID to request context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// WebSocketAuthMiddleware authenticates WebSocket connections
func WebSocketAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get database connection from context
		db, ok := r.Context().Value(DBKey).(*sql.DB)
		if !ok {
			utils.RespondWithError(w, http.StatusInternalServerError, "Database connection not found")
			return
		}

		// Check for session cookie first (for browser clients)
		sessionID, err := auth.GetSessionCookie(r)
		if err == nil {
			// Validate session
			userID, err := auth.ValidateSession(r.Context(), db, sessionID)
			if err == nil {
				// Add user ID to request context
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				next(w, r.WithContext(ctx))
				return
			}
		}

		// If cookie auth fails, try token auth (for mobile/non-browser clients)
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Also check for token in query parameters
			token := r.URL.Query().Get("token")
			if token != "" {
				authHeader = "Bearer " + token
			}
		}

		// If we have an auth header, try to validate it
		if authHeader != "" {
			// Check if the header has the correct format
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				// Validate token
				userID, err := auth.ValidateSession(r.Context(), db, parts[1])
				if err == nil {
					// Add user ID to request context
					ctx := context.WithValue(r.Context(), UserIDKey, userID)
					next(w, r.WithContext(ctx))
					return
				}
			}
		}

		// If all authentication methods fail, return unauthorized
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized: No valid authentication provided")
	}
}

// GetUserID extracts the user ID from the request context
func GetUserID(r *http.Request) (string, error) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		return "", errors.New("user ID not found in context")
	}
	return userID, nil
}

// DBMiddleware adds the database connection to the request context
func DBMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), DBKey, db)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
