package database

import (
	// context carries startup deadlines and cancellation to MongoDB.
	"context"
	// time sets a finite timeout for the connectivity check.
	"time"

	// mongo creates and represents the database client.
	"go.mongodb.org/mongo-driver/v2/mongo"
	// options constructs client settings from the connection URI.
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	// readpref selects the primary node for the startup ping.
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Connect creates a MongoDB client and checks that the database is reachable.
// The client is like a phone line: creating it is not proof anyone answered.
func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
	// Build a client configured with the supplied MongoDB connection string.
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		// Return setup errors so application startup can stop and report them.
		return nil, err
	}

	// Connect configures the client but does not prove the server is reachable,
	// so ping it and stop waiting after five seconds.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	// Release the timeout timer when the ping finishes.
	defer cancel()

	// Confirm that the primary is reachable before reporting a healthy connection.
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		// Propagate connectivity failures to the caller.
		return nil, err
	}
	// Return the ready-to-use client so callers can select a database.
	return client, nil
}
