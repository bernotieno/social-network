package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bernaotieno/social-network/backend/pkg/auth"
	"github.com/bernaotieno/social-network/backend/pkg/db/sqlite"
	"github.com/bernaotieno/social-network/backend/pkg/handlers"
	"github.com/bernaotieno/social-network/backend/pkg/middleware"
	"github.com/bernaotieno/social-network/backend/pkg/websocket"
	"github.com/gorilla/mux"
)

func main() {
	// Parse command line flags
	var (
		port           = flag.String("port", "8080", "Server port")
		dbPath         = flag.String("db", "./social_network.db", "SQLite database path")
		migrationsPath = flag.String("migrations", "./pkg/db/migrations/sqlite", "Path to migrations directory")
	)
	flag.Parse()

	// Initialize database
	db, err := sqlite.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := sqlite.RunMigrations(*dbPath, *migrationsPath); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations completed successfully")

	// Initialize auth package with a secret key
	// In production, this should be a secure random key stored in environment variables
	auth.Initialize([]byte("your-secret-key-here"))

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Create router
	r := mux.NewRouter()

	// Apply middleware
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.RecoveryMiddleware)
	r.Use(middleware.CORSMiddleware)
	r.Use(middleware.DBMiddleware(db))

	// Initialize handlers
	h := handlers.NewHandler(db, hub)

	// Register routes
	registerRoutes(r, h)
	// Handle all OPTIONS requests so CORS middleware runs
	r.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create server
	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Server shutting down...")
	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

func registerRoutes(r *mux.Router, h *handlers.Handler) {
	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Auth routes
	auth := api.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/register", h.Register).Methods("POST")
	auth.HandleFunc("/login", h.Login).Methods("POST")
	auth.HandleFunc("/logout", h.Logout).Methods("POST")

	// User routes
	users := api.PathPrefix("/users").Subrouter()
	users.HandleFunc("", h.GetUsers).Methods("GET")
	users.HandleFunc("/{id}", h.GetUser).Methods("GET")
	users.HandleFunc("/profile", middleware.AuthMiddleware(h.UpdateProfile)).Methods("PUT")
	users.HandleFunc("/avatar", middleware.AuthMiddleware(h.UploadAvatar)).Methods("POST")
	users.HandleFunc("/cover", middleware.AuthMiddleware(h.UploadCoverPhoto)).Methods("POST")
	users.HandleFunc("/{id}/follow", middleware.AuthMiddleware(h.FollowUser)).Methods("POST")
	users.HandleFunc("/{id}/follow", middleware.AuthMiddleware(h.UnfollowUser)).Methods("DELETE")
	users.HandleFunc("/{id}/followers", h.GetFollowers).Methods("GET")
	users.HandleFunc("/{id}/following", h.GetFollowing).Methods("GET")
	users.HandleFunc("/follow-requests", middleware.AuthMiddleware(h.GetFollowRequests)).Methods("GET")
	users.HandleFunc("/follow-requests/{id}", middleware.AuthMiddleware(h.RespondToFollowRequest)).Methods("PUT")

	// Post routes
	posts := api.PathPrefix("/posts").Subrouter()
	posts.HandleFunc("", middleware.AuthMiddleware(h.CreatePost)).Methods("POST")
	posts.HandleFunc("/feed", middleware.AuthMiddleware(h.GetFeed)).Methods("GET")
	posts.HandleFunc("/user/{id}", h.GetUserPosts).Methods("GET")
	posts.HandleFunc("/{id}", h.GetPost).Methods("GET")
	posts.HandleFunc("/{id}", middleware.AuthMiddleware(h.UpdatePost)).Methods("PUT")
	posts.HandleFunc("/{id}", middleware.AuthMiddleware(h.DeletePost)).Methods("DELETE")
	posts.HandleFunc("/{id}/like", middleware.AuthMiddleware(h.LikePost)).Methods("POST")
	posts.HandleFunc("/{id}/like", middleware.AuthMiddleware(h.UnlikePost)).Methods("DELETE")
	posts.HandleFunc("/{id}/comments", h.GetComments).Methods("GET")
	posts.HandleFunc("/{id}/comments", middleware.AuthMiddleware(h.AddComment)).Methods("POST")
	posts.HandleFunc("/{postId}/comments/{commentId}", middleware.AuthMiddleware(h.DeleteComment)).Methods("DELETE")

	// Group routes
	groups := api.PathPrefix("/groups").Subrouter()
	groups.HandleFunc("", middleware.AuthMiddleware(h.GetGroups)).Methods("GET")
	groups.HandleFunc("", middleware.AuthMiddleware(h.CreateGroup)).Methods("POST")
	groups.HandleFunc("/{id}", h.GetGroup).Methods("GET")
	groups.HandleFunc("/{id}", middleware.AuthMiddleware(h.UpdateGroup)).Methods("PUT")
	groups.HandleFunc("/{id}", middleware.AuthMiddleware(h.DeleteGroup)).Methods("DELETE")
	groups.HandleFunc("/{id}/join", middleware.AuthMiddleware(h.JoinGroup)).Methods("POST")
	groups.HandleFunc("/{id}/join", middleware.AuthMiddleware(h.LeaveGroup)).Methods("DELETE")
	groups.HandleFunc("/{id}/posts", h.GetGroupPosts).Methods("GET")
	groups.HandleFunc("/{id}/posts", middleware.AuthMiddleware(h.CreateGroupPost)).Methods("POST")
	groups.HandleFunc("/{id}/events", h.GetGroupEvents).Methods("GET")
	groups.HandleFunc("/{id}/events", middleware.AuthMiddleware(h.CreateGroupEvent)).Methods("POST")
	groups.HandleFunc("/events/{id}/respond", middleware.AuthMiddleware(h.RespondToEvent)).Methods("POST")

	// Notification routes
	notifications := api.PathPrefix("/notifications").Subrouter()
	notifications.HandleFunc("", middleware.AuthMiddleware(h.GetNotifications)).Methods("GET")
	notifications.HandleFunc("/{id}/read", middleware.AuthMiddleware(h.MarkNotificationAsRead)).Methods("PUT")
	notifications.HandleFunc("/read-all", middleware.AuthMiddleware(h.MarkAllNotificationsAsRead)).Methods("PUT")

	// WebSocket route
	r.HandleFunc("/ws", middleware.WebSocketAuthMiddleware(h.HandleWebSocket))

	// Static file server for uploaded images
	fs := http.FileServer(http.Dir("./uploads"))
	r.PathPrefix("/uploads/").Handler(middleware.CORSMiddleware(http.StripPrefix("/uploads/", fs)))
}
