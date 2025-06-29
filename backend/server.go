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
	"github.com/bernaotieno/social-network/backend/pkg/utils"
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

	// Initialize logger
	if logFile, err := utils.SetupLogFile(); err != nil {
		log.Fatalf("Failed to setup logger: %v", err)
	} else {
		defer logFile.Close()
	}
	// Initialize database
	db, err := sqlite.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	} else {
		utils.Logger("[INFO]: database created successfully", nil)
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

	// Create main router
	mainRouter := mux.NewRouter()

	// Initialize handlers
	h := handlers.NewHandler(db, hub)

	// Apply CORS middleware to ALL routes first
	mainRouter.Use(middleware.CORSMiddleware)

	// Register WebSocket route BEFORE applying other middleware
	mainRouter.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Add database to context
		ctx := context.WithValue(r.Context(), middleware.DBKey, db)
		r = r.WithContext(ctx)

		// Apply WebSocket auth middleware
		middleware.WebSocketAuthMiddleware(h.HandleWebSocket)(w, r)
	}).Methods("GET")

	// Create a subrouter for API routes with additional middleware
	apiRouter := mainRouter.PathPrefix("/api").Subrouter()
	apiRouter.Use(middleware.LoggingMiddleware)
	apiRouter.Use(middleware.RecoveryMiddleware)
	apiRouter.Use(middleware.DBMiddleware(db))

	// Register other routes on the API subrouter
	registerRoutes(apiRouter, h)

	// Static file server for uploaded images (on main router)
	fs := http.FileServer(http.Dir("./uploads"))
	mainRouter.PathPrefix("/uploads/").Handler(middleware.CORSMiddleware(http.StripPrefix("/uploads/", fs)))

	// Handle all OPTIONS requests so CORS middleware runs
	mainRouter.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create server
	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      mainRouter,
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

func registerRoutes(api *mux.Router, h *handlers.Handler) {
	// API routes are already prefixed with /api

	// Auth routes
	auth := api.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/register", h.Register).Methods("POST")
	auth.HandleFunc("/login", h.Login).Methods("POST")
	auth.HandleFunc("/logout", h.Logout).Methods("POST")

	// User routes
	users := api.PathPrefix("/users").Subrouter()
	users.HandleFunc("", middleware.AuthMiddleware(h.GetUsers)).Methods("GET")
	users.HandleFunc("/search", middleware.AuthMiddleware(h.GetUsers)).Methods("GET") // Reuse GetUsers for search functionality
	users.HandleFunc("/{id}", middleware.AuthMiddleware(h.GetUser)).Methods("GET")
	users.HandleFunc("/profile", middleware.AuthMiddleware(h.UpdateProfile)).Methods("PUT")
	users.HandleFunc("/avatar", middleware.AuthMiddleware(h.UploadAvatar)).Methods("POST")
	users.HandleFunc("/cover", middleware.AuthMiddleware(h.UploadCoverPhoto)).Methods("POST")
	users.HandleFunc("/{id}/follow", middleware.AuthMiddleware(h.FollowUser)).Methods("POST")
	users.HandleFunc("/{id}/follow", middleware.AuthMiddleware(h.UnfollowUser)).Methods("DELETE")
	users.HandleFunc("/{id}/followers", middleware.AuthMiddleware(h.GetFollowers)).Methods("GET")
	users.HandleFunc("/{id}/following", middleware.AuthMiddleware(h.GetFollowing)).Methods("GET")
	users.HandleFunc("/follow-requests", middleware.AuthMiddleware(h.GetFollowRequests)).Methods("GET")
	users.HandleFunc("/follow-requests/{id}", middleware.AuthMiddleware(h.RespondToFollowRequest)).Methods("PUT")

	// Post routes
	posts := api.PathPrefix("/posts").Subrouter()
	posts.HandleFunc("", middleware.AuthMiddleware(h.CreatePost)).Methods("POST")
	posts.HandleFunc("/feed", middleware.AuthMiddleware(h.GetFeed)).Methods("GET")
	posts.HandleFunc("/user/{id}", middleware.AuthMiddleware(h.GetUserPosts)).Methods("GET")
	posts.HandleFunc("/{id}", middleware.AuthMiddleware(h.GetPost)).Methods("GET")
	posts.HandleFunc("/{id}", middleware.AuthMiddleware(h.UpdatePost)).Methods("PUT")
	posts.HandleFunc("/{id}", middleware.AuthMiddleware(h.DeletePost)).Methods("DELETE")
	posts.HandleFunc("/{id}/like", middleware.AuthMiddleware(h.LikePost)).Methods("POST")
	posts.HandleFunc("/{id}/like", middleware.AuthMiddleware(h.UnlikePost)).Methods("DELETE")
	posts.HandleFunc("/{id}/comments", middleware.AuthMiddleware(h.GetComments)).Methods("GET")
	posts.HandleFunc("/{id}/comments", middleware.AuthMiddleware(h.AddComment)).Methods("POST")
	posts.HandleFunc("/{postId}/comments/{commentId}", middleware.AuthMiddleware(h.DeleteComment)).Methods("DELETE")

	// Group routes
	groups := api.PathPrefix("/groups").Subrouter()
	groups.HandleFunc("", middleware.AuthMiddleware(h.GetGroups)).Methods("GET")
	groups.HandleFunc("", middleware.AuthMiddleware(h.CreateGroup)).Methods("POST")
	groups.HandleFunc("/{id}", middleware.AuthMiddleware(h.GetGroup)).Methods("GET")
	groups.HandleFunc("/{id}", middleware.AuthMiddleware(h.UpdateGroup)).Methods("PUT")
	groups.HandleFunc("/{id}", middleware.AuthMiddleware(h.DeleteGroup)).Methods("DELETE")
	groups.HandleFunc("/{id}/join", middleware.AuthMiddleware(h.JoinGroup)).Methods("POST")
	groups.HandleFunc("/{id}/join", middleware.AuthMiddleware(h.LeaveGroup)).Methods("DELETE")
	groups.HandleFunc("/{id}/members", middleware.AuthMiddleware(h.GetGroupMembers)).Methods("GET")
	groups.HandleFunc("/{id}/members/{memberId}/promote", middleware.AuthMiddleware(h.PromoteGroupMember)).Methods("PUT")
	groups.HandleFunc("/{id}/members/{memberId}/demote", middleware.AuthMiddleware(h.DemoteGroupMember)).Methods("PUT")
	groups.HandleFunc("/{id}/members/{memberId}", middleware.AuthMiddleware(h.RemoveGroupMember)).Methods("DELETE")
	groups.HandleFunc("/{id}/pending-requests", middleware.AuthMiddleware(h.GetGroupPendingRequests)).Methods("GET")
	groups.HandleFunc("/{id}/approve-request", middleware.AuthMiddleware(h.ApproveJoinRequest)).Methods("POST")
	groups.HandleFunc("/{id}/reject-request", middleware.AuthMiddleware(h.RejectJoinRequest)).Methods("POST")
	groups.HandleFunc("/{id}/invite", middleware.AuthMiddleware(h.InviteToGroup)).Methods("POST")
	groups.HandleFunc("/invitations/{id}/respond", middleware.AuthMiddleware(h.RespondToGroupInvitation)).Methods("POST")
	groups.HandleFunc("/{id}/posts", middleware.AuthMiddleware(h.GetGroupPosts)).Methods("GET")
	groups.HandleFunc("/{id}/posts", middleware.AuthMiddleware(h.CreateGroupPost)).Methods("POST")
	groups.HandleFunc("/{groupId}/posts/{postId}", middleware.AuthMiddleware(h.DeleteGroupPost)).Methods("DELETE")
	groups.HandleFunc("/{groupId}/posts/{postId}/like", middleware.AuthMiddleware(h.LikeGroupPost)).Methods("POST")
	groups.HandleFunc("/{groupId}/posts/{postId}/like", middleware.AuthMiddleware(h.UnlikeGroupPost)).Methods("DELETE")
	groups.HandleFunc("/{groupId}/posts/{postId}/comments", middleware.AuthMiddleware(h.GetGroupPostComments)).Methods("GET")
	groups.HandleFunc("/{groupId}/posts/{postId}/comments", middleware.AuthMiddleware(h.AddGroupPostComment)).Methods("POST")
	groups.HandleFunc("/{groupId}/posts/{postId}/comments/{commentId}", middleware.AuthMiddleware(h.DeleteGroupPostComment)).Methods("DELETE")
	groups.HandleFunc("/{id}/events", middleware.AuthMiddleware(h.GetGroupEvents)).Methods("GET")
	groups.HandleFunc("/{id}/events", middleware.AuthMiddleware(h.CreateGroupEvent)).Methods("POST")
	groups.HandleFunc("/events/{id}", middleware.AuthMiddleware(h.UpdateGroupEvent)).Methods("PUT")
	groups.HandleFunc("/events/{id}", middleware.AuthMiddleware(h.DeleteGroupEvent)).Methods("DELETE")
	groups.HandleFunc("/events/{id}/respond", middleware.AuthMiddleware(h.RespondToEvent)).Methods("POST")
	groups.HandleFunc("/{id}/messages", middleware.AuthMiddleware(h.GetGroupMessages)).Methods("GET")
	groups.HandleFunc("/{id}/messages", middleware.AuthMiddleware(h.SendGroupMessage)).Methods("POST")

	// Notification routes
	notifications := api.PathPrefix("/notifications").Subrouter()
	notifications.HandleFunc("", middleware.AuthMiddleware(h.GetNotifications)).Methods("GET")
	notifications.HandleFunc("/read-all", middleware.AuthMiddleware(h.MarkAllNotificationsAsRead)).Methods("PUT")
	notifications.HandleFunc("/delete-all", middleware.AuthMiddleware(h.DeleteAllNotifications)).Methods("DELETE")
	notifications.HandleFunc("/{id}/read", middleware.AuthMiddleware(h.MarkNotificationAsRead)).Methods("PUT")
	notifications.HandleFunc("/{id}", middleware.AuthMiddleware(h.DeleteNotification)).Methods("DELETE")

	// Message routes
	messages := api.PathPrefix("/messages").Subrouter()
	messages.HandleFunc("", middleware.AuthMiddleware(h.SendMessage)).Methods("POST")
	messages.HandleFunc("/online-users", middleware.AuthMiddleware(h.GetOnlineUsers)).Methods("GET")
	messages.HandleFunc("/{userId}", middleware.AuthMiddleware(h.GetMessages)).Methods("GET")

	// WebSocket route is registered separately before middleware to avoid hijacker issues
	// Static file server is registered on the main router
}
