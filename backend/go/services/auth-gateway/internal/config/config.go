package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	OAuth    OAuthConfig
	Redis    RedisConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port string
	Host string
	Env  string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

// OAuthConfig holds OAuth provider configurations
type OAuthConfig struct {
	Google    OAuthProviderConfig
	Microsoft OAuthProviderConfig
	Apple     OAuthProviderConfig
}

// OAuthProviderConfig holds OAuth provider configuration
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Enabled      bool
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "serphona_auth"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			SecretKey:            getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			AccessTokenDuration:  parseDuration(getEnv("JWT_ACCESS_TOKEN_DURATION", "15m")),
			RefreshTokenDuration: parseDuration(getEnv("JWT_REFRESH_TOKEN_DURATION", "168h")), // 7 days
		},
		OAuth: OAuthConfig{
			Google: OAuthProviderConfig{
				ClientID:     getEnv("OAUTH_GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("OAUTH_GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("OAUTH_GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/oauth/google/callback"),
				Enabled:      getEnv("OAUTH_GOOGLE_ENABLED", "false") == "true",
			},
			Microsoft: OAuthProviderConfig{
				ClientID:     getEnv("OAUTH_MICROSOFT_CLIENT_ID", ""),
				ClientSecret: getEnv("OAUTH_MICROSOFT_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("OAUTH_MICROSOFT_REDIRECT_URL", "http://localhost:8080/api/v1/auth/oauth/microsoft/callback"),
				Enabled:      getEnv("OAUTH_MICROSOFT_ENABLED", "false") == "true",
			},
			Apple: OAuthProviderConfig{
				ClientID:     getEnv("OAUTH_APPLE_CLIENT_ID", ""),
				ClientSecret: getEnv("OAUTH_APPLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("OAUTH_APPLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/oauth/apple/callback"),
				Enabled:      getEnv("OAUTH_APPLE_ENABLED", "false") == "true",
			},
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
	}

	return config, nil
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseDuration parses duration string
func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 15 * time.Minute // default
	}
	return d
}
