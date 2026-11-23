// Package config loads and validates application configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// AppEnv identifies the application's runtime environment.
type AppEnv string

const (
	AppEnvDevelopment AppEnv = "development"
	AppEnvProduction  AppEnv = "production"
)

// Config contains the application's runtime configuration.
type Config struct {
	AppEnv               AppEnv
	Port                 string
	ClientURL            string
	DBURL                string
	JWTSecret            string
	JWTIssuer            string
	JWTAudience          string
	JWTAccessExpiration  time.Duration
	JWTRefreshExpiration time.Duration
	TogetherAPIKey       string
}

// Load parses and validates configuration from the environment and an optional
// .env file. Existing environment variables take precedence.
func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	appEnvValue, err := loadRequiredEnv("APP_ENV")
	if err != nil {
		return Config{}, err
	}
	appEnv := AppEnv(appEnvValue)
	if appEnv != AppEnvDevelopment && appEnv != AppEnvProduction {
		return Config{}, errors.New("invalid app environment")
	}

	portValue, err := loadRequiredEnv("PORT")
	if err != nil {
		return Config{}, err
	}
	portNumber, err := strconv.Atoi(portValue)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return Config{}, errors.New("invalid PORT")
	}
	port := strconv.Itoa(portNumber)

	clientURL, err := loadRequiredEnv("CLIENT_URL")
	if err != nil {
		return Config{}, err
	}
	dbURL, err := loadRequiredEnv("DB_URL")
	if err != nil {
		return Config{}, err
	}
	jwtSecret, err := loadRequiredEnv("JWT_SECRET")
	if err != nil {
		return Config{}, err
	}
	jwtIssuer, err := loadRequiredEnv("JWT_ISSUER")
	if err != nil {
		return Config{}, err
	}
	jwtAudience, err := loadRequiredEnv("JWT_AUDIENCE")
	if err != nil {
		return Config{}, err
	}
	jwtAccessExpirationValue, err := loadRequiredEnv("JWT_ACCESS_EXPIRES_IN")
	if err != nil {
		return Config{}, err
	}
	jwtAccessExpiration, err := time.ParseDuration(jwtAccessExpirationValue)
	if err != nil || jwtAccessExpiration <= 0 {
		return Config{}, errors.New("invalid JWT_ACCESS_EXPIRES_IN")
	}
	jwtRefreshExpirationValue, err := loadRequiredEnv("JWT_REFRESH_EXPIRES_IN")
	if err != nil {
		return Config{}, err
	}
	jwtRefreshExpiration, err := time.ParseDuration(jwtRefreshExpirationValue)
	if err != nil || jwtRefreshExpiration <= 0 {
		return Config{}, errors.New("invalid JWT_REFRESH_EXPIRES_IN")
	}

	togetherAPIKey, err := loadRequiredEnv("TOGETHER_API_KEY")
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppEnv:               appEnv,
		Port:                 port,
		ClientURL:            clientURL,
		DBURL:                dbURL,
		JWTSecret:            jwtSecret,
		JWTIssuer:            jwtIssuer,
		JWTAudience:          jwtAudience,
		JWTAccessExpiration:  jwtAccessExpiration,
		JWTRefreshExpiration: jwtRefreshExpiration,
		TogetherAPIKey:       togetherAPIKey,
	}, nil
}

// loadRequiredEnv returns a non-empty environment variable or an error.
func loadRequiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", errors.New("missing required environment variable: " + name)
	}
	return value, nil
}
