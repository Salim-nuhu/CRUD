package main

import (
	// context lets startup and shutdown operations carry cancellation information.
	"context"
	// log writes startup messages and terminates the program on fatal errors.
	"log"
	// http provides the server and HTTP handler types.
	"net/http"
	// os provides operating-system signals and process support.
	"os"
	// signal delivers interrupt notifications to the shutdown channel.
	"os/signal"
	// time defines server timeouts and the graceful-shutdown deadline.
	"time"

	// auth creates and manages signed authentication tokens.
	"github.com/Salim-nuhu/CRUD/internal/auth"
	// config reads application settings from environment variables and .env.
	"github.com/Salim-nuhu/CRUD/internal/config"
	// database establishes the MongoDB connection used by the API.
	"github.com/Salim-nuhu/CRUD/internal/database"
	// handlers implement the HTTP endpoints.
	"github.com/Salim-nuhu/CRUD/internal/handlers"
	// middleware adds request logging and panic recovery around routes.
	"github.com/Salim-nuhu/CRUD/internal/middleware"
	// repository contains the MongoDB-backed user data operations.
	"github.com/Salim-nuhu/CRUD/internal/repository"
	// router registers URL patterns and connects them to handlers.
	"github.com/Salim-nuhu/CRUD/internal/router"

	"github.com/Salim-nuhu/CRUD/internal/models"
)

// main loads configuration, assembles the API, and controls its server lifecycle.
func main() {
	// Load settings before constructing services that depend on them.
	cfg := config.Load()

	// MongoDB is required, so stop early with a useful configuration hint.
	if cfg.MongoURI == "" {
		log.Fatal("MONGO_URI is not set (check your .env file)")
	}
	// Require a sufficiently long signing key rather than using a weak default.
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be set and at least 32 characters (check your .env file)")
	}

	// Connect to MongoDB using a background context for startup.
	client, err := database.Connect(context.Background(), cfg.MongoURI)
	if err != nil {
		log.Fatalf("could not connect to MongoDB: %v", err)
	}
	// Close the MongoDB client when main exits.
	defer client.Disconnect(context.Background())
	// Record successful database initialization.
	log.Println("connected to MongoDB")

	// Select the configured database from the connected client.
	db := client.Database(cfg.DBName)

	// Create the user repository, including its required database index.
	userRepo, err := repository.NewUserRepository(db)
	if err != nil {
		log.Fatalf("could not set up user repository: %v", err)
	}

	// Create tokens that expire one hour after issuance.
	tokens := auth.NewTokenManager(cfg.JWTSecret, time.Hour)

	// Bind user operations to the shared repository.
	userHandler := handlers.NewUserHandler(userRepo)
	// Bind login operations to the repository and token manager.
	authHandler := handlers.NewAuthHandler(userRepo, tokens)

	// Logger is outermost, so it also sees the 500 that Recover produces.
	// Wrap routes with request logging and panic recovery.
	protect := middleware.RequireAuth(tokens)
	ownerOnly := middleware.RequireRole(models.RoleOwner)

	// Logger is outermost so every request is logged. CORS comes before
	// Recover and the router so that even error responses carry CORS
	// headers: otherwise a browser would hide a real 401 behind a CORS error.
	handler := middleware.Chain(
		router.New(userHandler, authHandler, protect, ownerOnly),
		middleware.Logger,
		middleware.CORS(cfg.AllowedOrigins),
		middleware.Recover,
	)

	// Configure the HTTP listener, application handler, and request timeouts.
	srv := &http.Server{
		// Listen on the configured TCP port on all network interfaces.
		Addr: ":" + cfg.Port,
		// Send incoming requests through the configured middleware and routes.
		Handler: handler,
		// Limit how long the server waits to read a request.
		ReadTimeout: 10 * time.Second,
		// Limit how long the server waits to write a response.
		WriteTimeout: 10 * time.Second,
	}

	// Run the blocking HTTP listener in the background so main can await a signal.
	go func() {
		// Announce the address before accepting requests.
		log.Printf("server listening on :%s", cfg.Port)
		// Ignore the normal close error but report unexpected listener failures.
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Buffer one interrupt signal so notification does not block the sender.
	stop := make(chan os.Signal, 1)
	// Ask Go to deliver Ctrl+C or equivalent interrupt signals to stop.
	signal.Notify(stop, os.Interrupt)
	// Pause startup goroutine until an interrupt arrives.
	<-stop

	// Allow active requests up to five seconds to finish during shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// Release the timeout timer when shutdown completes.
	defer cancel()
	// Stop accepting new connections and gracefully finish active requests.
	srv.Shutdown(ctx)
	// Announce that the server has stopped.
	log.Println("server stopped")
}
