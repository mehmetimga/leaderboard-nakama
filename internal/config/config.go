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
	Kafka    KafkaConfig
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

// KafkaConfig holds Kafka connection settings
type KafkaConfig struct {
	Enabled bool
	Brokers []string
	Topic   string
	GroupID string
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
		Kafka: KafkaConfig{
			Enabled: getBool("KAFKA_ENABLED", false),
			Brokers: getStringSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			Topic:   getEnv("KAFKA_TOPIC", "leaderboard-scores"),
			GroupID: getEnv("KAFKA_GROUP_ID", "leaderboard-consumer"),
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

func getStringSlice(key string, defaultVal []string) []string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	var result []string
	for _, s := range splitString(val, ',') {
		trimmed := trimString(s)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s string, sep byte) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimString(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
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

