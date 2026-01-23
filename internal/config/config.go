package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database OracleConfig
	JWT      JWTConfig
	Auth     AuthConfig
	Print    PrintConfig
	LogLevel string
}

// PrintConfig holds print service configuration
type PrintConfig struct {
	OutputPath  string
	JobInterval time.Duration
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	MaxHeaderBytes  int
	ShutdownTimeout time.Duration
}

// JWTConfig holds JWT-related configuration
type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

// AuthConfig holds authentication service configuration
type AuthConfig struct {
	BaseURL string
}

// Load loads configuration from environment variables
// Panics if required configuration is missing
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
			Port:            getEnvOrDefault("SERVER_PORT", "8080"),
			ReadTimeout:     getDurationOrDefault("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDurationOrDefault("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getDurationOrDefault("SERVER_IDLE_TIMEOUT", 60*time.Second),
			MaxHeaderBytes:  getIntOrDefault("SERVER_MAX_HEADER_BYTES", 1<<20), // 1MB default
			ShutdownTimeout: getDurationOrDefault("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: OracleConfig{
			Host:         getEnvOrDefault("ORACLE_HOST", "localhost"),
			Port:         getEnvOrDefault("ORACLE_PORT", "1521"),
			Service:      getEnvOrDefault("ORACLE_SERVICE", "ORCL"),
			User:         os.Getenv("ORACLE_USER"),
			Password:     os.Getenv("ORACLE_PASSWORD"),
			MaxOpenConns: getIntOrDefault("ORACLE_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getIntOrDefault("ORACLE_MAX_IDLE_CONNS", 5),
			WalletPath:   os.Getenv("ORACLE_WALLET_PATH"),
			TNSAlias:     os.Getenv("ORACLE_TNS_ALIAS"),
		},
		JWT: JWTConfig{
			Secret:     requireEnv("JWT_SECRET"),
			Expiration: getDurationOrDefault("JWT_EXPIRATION", 24*time.Hour),
		},
		Auth: AuthConfig{
			BaseURL: getEnvOrDefault("AUTH_SERVICE_URL", "http://localhost:8081"),
		},
		Print: PrintConfig{
			OutputPath:  getEnvOrDefault("PRINT_OUTPUT_PATH", "./output"),
			JobInterval: getDurationOrDefault("PRINT_JOB_INTERVAL", 30*time.Second),
		},
		LogLevel: getEnvOrDefault("LOG_LEVEL", "info"),
	}
}

// requireEnv returns the value of the environment variable or panics if not set
func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return val
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getIntOrDefault(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
