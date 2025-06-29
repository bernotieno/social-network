package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/bernaotieno/social-network/backend/pkg/utils"
)

// RecoveryMiddleware recovers from panics and returns a 500 error
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the error and stack trace
				log.Printf("PANIC: %v\n%s", err, debug.Stack())

				// Return a 500 Internal Server Error
				utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
