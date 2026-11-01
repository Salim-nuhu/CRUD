package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Salim-nuhu/CRUD/internal/config"
	"github.com/Salim-nuhu/CRUD/internal/database"
	"github.com/Salim-nuhu/CRUD/internal/handlers"
	"github.com/Salim-nuhu/CRUD/internal/repository"
)

// main connects the app's pieces and starts the HTTP server.
func main() {
	// Load the port and MongoDB settings before creating any services.
	cfg := config.Load()

	// Connecting with an empty address cannot work, so report the missing
	// setting immediately rather than failing later with a confusing error.
	if cfg.MongoURI == "" {
		log.Fatal("MONGO_URI is not set (check your .env file)")
	}

	// A context is a timer/cancellation signal shared with the connection code.
	client, err := database.Connect(context.Background(), cfg.MongoURI)
	if err != nil {
		log.Fatalf("could not connect to MongoDB: %v", err)
	}
	// Close the database connection when the program exits.
	defer client.Disconnect(context.Background())
	log.Println("connected to MongoDB")

	// Select the configured database, then prepare the user storage and HTTP
	// handler. The handler handles web requests; the repository handles MongoDB.
	db := client.Database(cfg.DBName)

	userRepo, err := repository.NewUserRepository(db)
	if err != nil {
		log.Fatalf("could not set up user repository: %v", err)
	}
	userHandler := handlers.NewUserHandler(userRepo)

	// The mux is a traffic director: it sends each URL and method to its handler.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		// A small health response lets tools check whether the API is running.
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	// Route requests that create a user to the user-specific handler.
	mux.HandleFunc("POST /users", userHandler.Create)

	// Timeouts prevent a slow client from keeping a connection busy forever.
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Run the server in the background so this function can also wait for Ctrl+C.
	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for Ctrl+C, then give requests already in progress time to finish.
	// This is like closing a shop to new customers while serving the people inside.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	// Bound shutdown time so a stuck request cannot prevent the app from exiting.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("server stopped")
}
