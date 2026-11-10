package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	Port           string
	MongoURI       string
	DBName         string
	JWTSecret      string
	AllowedOrigins []string
}

func Load() Config {
	loadDotEnv(".env")

	return Config{
		Port:      getEnv("PORT", "8080"),
		MongoURI:  getEnv("MONGO_URI", ""),
		DBName:    getEnv("DB_NAME", "crud_api"),
		JWTSecret: getEnv("JWT_SECRET", ""),
		// Origins allowed to call the API from a browser. The defaults are
		// the usual local dev servers; set ALLOWED_ORIGINS to override.
		AllowedOrigins: splitList(getEnv("ALLOWED_ORIGINS",
			"http://localhost:5500,http://127.0.0.1:5500,http://localhost:3000")),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// splitList turns "a, b ,c" into ["a" "b" "c"], dropping empty items.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// loadDotEnv reads KEY=VALUE lines from a file into the environment.
// Real environment variables win, so production can override the file.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // no .env file is fine
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}
