package config

import (
	// bufio scans the .env file one line at a time.
	"bufio"
	// os reads process environment variables and opens the optional file.
	"os"
	// strings trims and separates each KEY=VALUE entry.
	"strings"
)

// Config contains the runtime settings used to assemble the application.
type Config struct {
	// Port is the TCP port where the HTTP server listens.
	Port string
	// MongoURI is the connection string for the MongoDB deployment.
	MongoURI string
	// DBName selects the database containing application collections.
	DBName string
	// JWTSecret signs authentication tokens and must be kept private.
	JWTSecret string
}

// Load reads optional .env values and returns settings with safe defaults.
func Load() Config {
	// Import local settings without overriding existing process environment values.
	loadDotEnv(".env")

	// Resolve each setting, leaving the secret empty so startup must validate it.
	return Config{
		// Use a conventional development port when none is configured.
		Port: getEnv("PORT", "8080"),
		// Require the MongoDB URI to be provided explicitly.
		MongoURI: getEnv("MONGO_URI", ""),
		// Use a predictable database name unless deployment config overrides it.
		DBName:    getEnv("DB_NAME", "crud_api"),
		JWTSecret: getEnv("JWT_SECRET", ""), // no default, on purpose
	}
}

// getEnv returns a non-empty environment value or its supplied fallback.
func getEnv(key, fallback string) string {
	// Existing process settings take precedence over values loaded from a file.
	if v := os.Getenv(key); v != "" {
		return v
	}
	// Return the fallback when the environment key is missing or empty.
	return fallback
}

// loadDotEnv reads KEY=VALUE lines from a file into the environment.
// Real environment variables win, so production can override the file.
func loadDotEnv(path string) {
	// Open the optional configuration file; absence is allowed.
	f, err := os.Open(path)
	if err != nil {
		return // no .env file is fine
	}
	// Close the file after its values have been read.
	defer f.Close()

	// Create a scanner that advances through the file one line at a time.
	scanner := bufio.NewScanner(f)
	// The outer scan advances once before the inner loop processes later lines.
	for scanner.Scan() {
		// Continue scanning and parsing lines until the scanner reaches the end.
		for scanner.Scan() {
			// Remove surrounding whitespace and an optional UTF-8 byte-order mark.
			line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
			// Ignore blank lines and lines whose first non-space character is #.
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Split once at the first equals sign so values may contain equals signs.
			key, value, found := strings.Cut(line, "=")
			// Ignore malformed lines that do not contain a separator.
			if !found {
				continue
			}
			// Normalize the key and remove whitespace and matching quote characters.
			key = strings.TrimSpace(key)
			value = strings.Trim(strings.TrimSpace(value), `"'`)

			// Preserve environment values already set by the process or deployment.
			if _, exists := os.LookupEnv(key); !exists {
				// Add this file value only when the process has no value for the key.
				os.Setenv(key, value)
			}
		}
	}
}
