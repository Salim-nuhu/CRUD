package config

import (
	"os"
	"path/filepath"
	"strings"
)

// Config is the small set of settings the API needs to start.
// Think of it like a labeled card with the server and database addresses.
type Config struct {
	Port     string
	MongoURI string
	DBName   string
}

// Load reads settings from the environment, using .env as a convenient local
// source. Environment variables take priority so deployed servers can provide
// their own settings without changing a file.
func Load() Config {
	loadDotEnv(".env")

	return Config{
		Port:     getEnv("PORT", "8080"),
		MongoURI: getEnv("MONGO_URI", ""),
		DBName:   getEnv("DB_NAME", "crud_api"),
	}
}

// loadDotEnv reads simple KEY=value lines and places them in the process
// environment, where getEnv can read them just like settings set by the OS.
func loadDotEnv(path string) {
	candidates := []string{path}
	if !filepath.IsAbs(path) {
		// The program might be started from the project folder or a child folder,
		// so check both locations like looking in nearby drawers for the file.
		candidates = append(candidates,
			filepath.Join(".", path),
			filepath.Join("..", path),
		)
	}

	for _, candidate := range candidates {
		contents, err := os.ReadFile(candidate)
		if err != nil {
			// This location may not contain the file; try the next candidate.
			continue
		}

		for _, line := range strings.Split(string(contents), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				// Blank lines and comments are notes for people, not settings.
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				// A setting needs both a name and a value, separated by "=".
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			// Do not replace a value already supplied by the operating system.
			// That lets production settings override local .env defaults.
			_ = os.Setenv(key, value)
		}
		// Use the first .env file found instead of loading multiple copies.
		return
	}
}

// getEnv returns a configured value, or a safe default when it is missing.
// This is like using a spare key when the main key was not provided.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
