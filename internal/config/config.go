package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Nakama   NakamaConfig
	Postgres PostgresConfig
	Worker   WorkerConfig
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// NakamaConfig holds Nakama server connection settings
type NakamaConfig struct {
	Host       string
	GRPCPort   string
	HTTPPort   string
	ServerKey  string
	UseSSL     bool
}

// PostgresConfig holds PostgreSQL connection settings
type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

// WorkerConfig holds background worker settings
type WorkerConfig struct {
	SnapshotInterval time.Duration
	Enabled          bool
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		},
		Nakama: NakamaConfig{
			Host:      getEnv("NAKAMA_HOST", "localhost"),
			GRPCPort:  getEnv("NAKAMA_GRPC_PORT", "7349"),
			HTTPPort:  getEnv("NAKAMA_HTTP_PORT", "7350"),
			ServerKey: getEnv("NAKAMA_SERVER_KEY", "defaultkey"),
			UseSSL:    getBool("NAKAMA_USE_SSL", false),
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "postgres"),
			Database: getEnv("POSTGRES_DB", "leaderboard"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		Worker: WorkerConfig{
			SnapshotInterval: getDuration("WORKER_SNAPSHOT_INTERVAL", 30*time.Minute),
			Enabled:          getBool("WORKER_ENABLED", true),
		},
	}, nil
}

// DSN returns the PostgreSQL connection string
func (c *PostgresConfig) DSN() string {
	return "postgres://" + c.User + ":" + c.Password + "@" + c.Host + ":" + c.Port + "/" + c.Database + "?sslmode=" + c.SSLMode
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}

func getDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultVal
	}
	return d
}

