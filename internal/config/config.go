// Package config
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Env              string // "dev" | "prod"
	LocalStorage     string
	Port             string
	SizeLimitMB      int64
	AsynqConcurrency int

	// Postgres settings
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis settings
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// Load reads configuration from the process environment, falling back to a
// local .env file for development. It fails when a required setting is missing
// rather than starting up half configured.
func Load() (*Config, error) {
	err := godotenv.Load()

	if err != nil {
		log.Print("Unable to load .env file")
	}

	var missing []string
	for _, name := range []string{"DB_HOST", "DB_USER", "DB_NAME", "DB_SSLMODE", "REDIS_ADDR"} {
		if os.Getenv(name) == "" {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s (see .env.example)",
			strings.Join(missing, ", "))
	}

	// An empty password is allowed for local environment testing
	if os.Getenv("DB_PASSWORD") == "" {
		log.Print("DB_PASSWORD is empty: connecting to Postgres without a password")
	}

	if os.Getenv("REDIS_PASSWORD") == "" {
		log.Print("REDIS_PASSWORD is empty: connecting to Redis without a password")
	}

	sizeLimitMB, err := strconv.ParseInt(getEnv("SIZE_LIMIT_MB", "512"), 10, 64)
	if err != nil {
		log.Print("Failed to convert SIZE_LIMIT_MB, setting the value to default")
		sizeLimitMB = 512
	}

	// ensure that the absolute storage path is used
	storageDir := getEnv("STORAGE_DIR", "./tmp/")
	absPath, err := filepath.Abs(storageDir)
	if err == nil {
		storageDir = absPath
	}

	asynqConcurrency, err := strconv.Atoi(getEnv("ASYNQ_CONCURRENCY", "2"))
	if err != nil {
		log.Print("Failed to convert ASYNQ_CONCURRENCY, setting the value to default")
		asynqConcurrency = 2
	}

	redisDB := 0
	if v := os.Getenv("REDIS_DB"); v != "" {
		redisDB, err = strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB %q: %w", v, err)
		}
	}

	return &Config{
		Env:              getEnv("APP_DEV", "dev"),
		LocalStorage:     storageDir,
		Port:             getEnv("PORT", "3000"),
		SizeLimitMB:      sizeLimitMB,
		AsynqConcurrency: asynqConcurrency,
		DBHost:           os.Getenv("DB_HOST"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           os.Getenv("DB_USER"),
		DBPassword:       os.Getenv("DB_PASSWORD"),
		DBName:           os.Getenv("DB_NAME"),
		DBSSLMode:        os.Getenv("DB_SSLMODE"),
		RedisAddr:        os.Getenv("REDIS_ADDR"),
		RedisPassword:    os.Getenv("REDIS_PASSWORD"),
		RedisDB:          redisDB,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
