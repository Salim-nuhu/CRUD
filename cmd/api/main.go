package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Salim-nuhu/CRUD/internal/auth"
	"github.com/Salim-nuhu/CRUD/internal/config"
	"github.com/Salim-nuhu/CRUD/internal/database"
	"github.com/Salim-nuhu/CRUD/internal/handlers"
	"github.com/Salim-nuhu/CRUD/internal/middleware"
	"github.com/Salim-nuhu/CRUD/internal/models"
	"github.com/Salim-nuhu/CRUD/internal/repository"
	"github.com/Salim-nuhu/CRUD/internal/router"
)

func main() {
	cfg := config.Load()

	if cfg.MongoURI == "" {
		log.Fatal("MONGO_URI is not set (check your .env file)")
	}
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be set and at least 32 characters (check your .env file)")
	}

	client, err := database.Connect(context.Background(), cfg.MongoURI)
	if err != nil {
		log.Fatalf("could not connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())
	log.Println("connected to MongoDB")

	db := client.Database(cfg.DBName)

	userRepo, err := repository.NewUserRepository(db)
	if err != nil {
		log.Fatalf("could not set up user repository: %v", err)
	}
	propertyRepo, err := repository.NewPropertyRepository(db)
	if err != nil {
		log.Fatalf("could not set up property repository: %v", err)
	}

	tokens := auth.NewTokenManager(cfg.JWTSecret, time.Hour)

	userHandler := handlers.NewUserHandler(userRepo)
	authHandler := handlers.NewAuthHandler(userRepo, tokens)
	propertyHandler := handlers.NewPropertyHandler(propertyRepo)

	protect := middleware.RequireAuth(tokens)
	optional := middleware.OptionalAuth(tokens)
	ownerOnly := middleware.RequireRole(models.RoleOwner)

	// Logger is outermost so every request is logged. CORS comes before
	// Recover and the router so that even error responses carry CORS headers.
	handler := middleware.Chain(
		router.New(userHandler, authHandler, propertyHandler, protect, optional, ownerOnly),
		middleware.Logger,
		middleware.CORS(cfg.AllowedOrigins),
		middleware.Recover,
	)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("server stopped")
}
