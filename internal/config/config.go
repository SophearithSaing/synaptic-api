// Package config loads and validates process configuration from the
// environment, with optional dotenv file support for local development.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Config holds the validated process configuration.
type Config struct {
	// AppEnv is the deployment environment, e.g. "development" or
	// "production".
	AppEnv string
	// Port is the HTTP listen port.
	Port int
	// ClientURL is the single credentialed CORS origin.
	ClientURL string
	// MongoURI is the MongoDB connection string.
	MongoURI string
	// MongoDatabase is the MongoDB database name.
	MongoDatabase string
	// JWTSecret signs and verifies access tokens.
	JWTSecret string
	// JWTIssuer is the required JWT iss claim.
	JWTIssuer string
	// JWTAudience is the required JWT aud claim.
	JWTAudience string
	// JWTAccessTTL is the access token lifetime.
	JWTAccessTTL time.Duration
	// JWTRefreshTTL is the refresh token lifetime.
	JWTRefreshTTL time.Duration
	// TogetherAPIKey authenticates with the Together AI API.
	TogetherAPIKey string
	// ThrottleTrustedProxies lists the reverse-proxy addresses or
	// CIDRs whose X-Forwarded-For values the rate limiter trusts.
	// Empty keeps direct transport-peer throttling.
	ThrottleTrustedProxies []string
}

// MongoConfig holds the database settings required by maintenance commands.
type MongoConfig struct {
	// URI is the MongoDB connection string.
	URI string
	// Database is the MongoDB database name.
	Database string
}

// Production reports whether the process runs in production mode.
func (c Config) Production() bool {
	return c.AppEnv == "production"
}

// SecureCookies reports whether auth cookies use the production policy
// (Secure with SameSite=None). Other environments use SameSite=Lax without
// Secure, because browsers reject SameSite=None cookies over plain HTTP.
func (c Config) SecureCookies() bool {
	return c.Production()
}

// Load reads an optional .env file and returns the validated configuration.
// Every required variable must be present or the process fails at startup.
func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:    envOr("APP_ENV", "development"),
		ClientURL: envOr("CLIENT_URL", "http://localhost:4200"),
	}

	port, err := strconv.Atoi(envOr("PORT", "3000"))
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be a valid TCP port number")
	}
	cfg.Port = port

	var errs []error
	cfg.MongoURI = required("DB_URI", &errs)
	cfg.MongoDatabase = required("DB_NAME", &errs)
	cfg.JWTSecret = required("JWT_SECRET", &errs)
	cfg.JWTIssuer = required("JWT_ISSUER", &errs)
	cfg.JWTAudience = required("JWT_AUDIENCE", &errs)
	cfg.TogetherAPIKey = required("TOGETHER_API_KEY", &errs)
	cfg.JWTAccessTTL = durationEnv("JWT_ACCESS_EXPIRES_IN", &errs)
	cfg.JWTRefreshTTL = durationEnv("JWT_REFRESH_EXPIRES_IN", &errs)
	trustedProxies, err := trustedProxies()
	if err != nil {
		errs = append(errs, fmt.Errorf("THROTTLE_TRUSTED_PROXIES: %w", err))
	}
	cfg.ThrottleTrustedProxies = trustedProxies

	return cfg, errors.Join(errs...)
}

// LoadMongo reads the MongoDB settings without requiring API-only settings.
func LoadMongo() (MongoConfig, error) {
	if err := loadDotEnv(".env"); err != nil {
		return MongoConfig{}, err
	}

	var errs []error
	cfg := MongoConfig{
		URI:      required("DB_URI", &errs),
		Database: required("DB_NAME", &errs),
	}

	return cfg, errors.Join(errs...)
}

// trustedProxies parses the optional comma-separated
// THROTTLE_TRUSTED_PROXIES list of exact addresses or CIDR ranges,
// rejecting unparseable entries.
func trustedProxies() ([]string, error) {
	raw := os.Getenv("THROTTLE_TRUSTED_PROXIES")
	if raw == "" {
		return nil, nil
	}

	var proxies []string
	for _, entry := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		proxies = append(proxies, trimmed)
	}

	return proxies, validateTrustedProxies(proxies)
}

// validateTrustedProxies reports whether every trusted proxy entry is
// an exact IP address or a CIDR range.
func validateTrustedProxies(proxies []string) error {
	var errs []error
	for _, proxy := range proxies {
		if strings.Contains(proxy, "/") {
			if _, _, err := net.ParseCIDR(proxy); err != nil {
				errs = append(errs, fmt.Errorf(
					"trusted proxy %q: %w", proxy, err,
				))
			}

			continue
		}
		if net.ParseIP(proxy) == nil {
			errs = append(errs, fmt.Errorf(
				"trusted proxy %q: want an exact IP address or CIDR range",
				proxy,
			))
		}
	}

	return errors.Join(errs...)
}

var durationPattern = regexp.MustCompile(`^(\d+)(ms|s|m|h|d)$`)

// ParseDuration converts a compact single-unit duration string such as
// "15m" or "7d" into a time.Duration, matching the legacy syntax.
func ParseDuration(value string) (time.Duration, error) {
	match := durationPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return 0, fmt.Errorf(
			"invalid duration %q: want a number followed by ms, s, m, h, or d",
			value,
		)
	}

	amount, _ := strconv.Atoi(match[1])
	switch match[2] {
	case "ms":
		return time.Duration(amount) * time.Millisecond, nil
	case "s":
		return time.Duration(amount) * time.Second, nil
	case "m":
		return time.Duration(amount) * time.Minute, nil
	case "h":
		return time.Duration(amount) * time.Hour, nil
	case "d":
		return time.Duration(amount) * 24 * time.Hour, nil
	}

	return 0, fmt.Errorf("invalid duration unit %q", match[2])
}

// envOr returns the environment variable value or the fallback when unset.
func envOr(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

// required returns a mandatory environment variable or records it missing.
func required(key string, errs *[]error) string {
	value := os.Getenv(key)
	if value == "" {
		*errs = append(*errs, fmt.Errorf("%s is required", key))
	}

	return value
}

// durationEnv returns a mandatory compact duration environment variable.
func durationEnv(key string, errs *[]error) time.Duration {
	raw := required(key, errs)
	if raw == "" {
		return 0
	}

	duration, err := ParseDuration(raw)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %w", key, err))
		return 0
	}

	return duration
}

// loadDotEnv adds variables from a dotenv file to the environment without
// overriding variables that are already set. A missing file is not an error.
func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}

	return scanner.Err()
}
