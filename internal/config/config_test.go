package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/config"
)

func TestParseDuration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{name: "milliseconds", value: "500ms", want: 500 * time.Millisecond},
		{name: "seconds", value: "30s", want: 30 * time.Second},
		{name: "minutes", value: "15m", want: 15 * time.Minute},
		{name: "hours", value: "12h", want: 12 * time.Hour},
		{name: "days", value: "7d", want: 7 * 24 * time.Hour},
		{name: "surrounding space", value: " 7d ", want: 7 * 24 * time.Hour},
		{name: "empty", value: "", wantErr: true},
		{name: "missing unit", value: "10", wantErr: true},
		{name: "unknown unit", value: "10x", wantErr: true},
		{name: "compound", value: "1h30m", wantErr: true},
		{name: "negative", value: "-5m", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := config.ParseDuration(tc.value)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q", tc.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// setRequiredEnv sets every required variable to a valid value.
func setRequiredEnv(t *testing.T) {
	t.Helper()

	t.Setenv("DB_URI", "mongodb://localhost:27017")
	t.Setenv("DB_NAME", "synaptic")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_ISSUER", "synaptic-api")
	t.Setenv("JWT_AUDIENCE", "synaptic-web")
	t.Setenv("JWT_ACCESS_EXPIRES_IN", "15m")
	t.Setenv("JWT_REFRESH_EXPIRES_IN", "7d")
	t.Setenv("TOGETHER_API_KEY", "test-key")
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 3000 {
		t.Errorf("got port %d, want 3000", cfg.Port)
	}
	if cfg.ClientURL != "http://localhost:4200" {
		t.Errorf("got client URL %q", cfg.ClientURL)
	}
	if cfg.AppEnv != "development" {
		t.Errorf("got app env %q", cfg.AppEnv)
	}
	if cfg.SecureCookies() {
		t.Error("secure cookies must be disabled outside production")
	}
	if cfg.JWTAccessTTL != 15*time.Minute {
		t.Errorf("got access TTL %v", cfg.JWTAccessTTL)
	}
	if cfg.JWTRefreshTTL != 7*24*time.Hour {
		t.Errorf("got refresh TTL %v", cfg.JWTRefreshTTL)
	}
}

func TestLoadMongoOnlyRequiresDatabaseSettings(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DB_URI", "mongodb://localhost:27017")
	t.Setenv("DB_NAME", "synaptic")
	t.Setenv("JWT_SECRET", "")

	cfg, err := config.LoadMongo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.URI != "mongodb://localhost:27017" {
		t.Errorf("got URI %q", cfg.URI)
	}
	if cfg.Database != "synaptic" {
		t.Errorf("got database %q", cfg.Database)
	}
}

func TestLoadProductionEnablesSecureCookies(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("APP_ENV", "production")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.SecureCookies() {
		t.Error("secure cookies must be enabled in production")
	}
}

func TestLoadReportsMissingRequired(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, key := range []string{
		"DB_URI", "DB_NAME", "JWT_SECRET", "JWT_ISSUER", "JWT_AUDIENCE",
		"JWT_ACCESS_EXPIRES_IN", "JWT_REFRESH_EXPIRES_IN", "TOGETHER_API_KEY",
	} {
		t.Setenv(key, "")
	}

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected an error for missing variables")
	}
	for _, key := range []string{"DB_URI", "JWT_SECRET", "TOGETHER_API_KEY"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not mention %s", err, key)
		}
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("PORT", "not-a-port")

	if _, err := config.Load(); err == nil {
		t.Fatal("expected an error for an invalid port")
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("JWT_ACCESS_EXPIRES_IN", "1h30m")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected an error for a compound duration")
	}
	if !strings.Contains(err.Error(), "JWT_ACCESS_EXPIRES_IN") {
		t.Errorf("error %q does not name the variable", err)
	}
}

func TestLoadDefaultsToNoTrustedProxies(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ThrottleTrustedProxies != nil {
		t.Errorf("default trusted proxies %+v, want nil", cfg.ThrottleTrustedProxies)
	}
}

func TestLoadParsesTrustedProxies(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("THROTTLE_TRUSTED_PROXIES",
		" 192.0.2.1 , 10.0.0.0/8 ,\t172.16.0.1\t")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"192.0.2.1", "10.0.0.0/8", "172.16.0.1"}
	if len(cfg.ThrottleTrustedProxies) != len(want) {
		t.Fatalf("parsed %+v, want %+v", cfg.ThrottleTrustedProxies, want)
	}
	for index, proxy := range want {
		if cfg.ThrottleTrustedProxies[index] != proxy {
			t.Errorf("entry %d %q, want %q",
				index, cfg.ThrottleTrustedProxies[index], proxy)
		}
	}
}

func TestLoadRejectsInvalidTrustedProxies(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("THROTTLE_TRUSTED_PROXIES", "10.0.0.1,not-a-proxy")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected an error for an invalid trusted proxy")
	}
	if !strings.Contains(err.Error(), "THROTTLE_TRUSTED_PROXIES") {
		t.Errorf("error %q does not name THROTTLE_TRUSTED_PROXIES", err)
	}
}

func TestLoadReadsDotEnvWithoutOverriding(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	dotenv := "# comment\n" +
		"DB_URI=mongodb://from-file:27017\n" +
		"DB_NAME=from-file\n" +
		"JWT_SECRET=file-secret\n" +
		"JWT_ISSUER=file-issuer\n" +
		"JWT_AUDIENCE=file-audience\n" +
		"JWT_ACCESS_EXPIRES_IN=15m\n" +
		"JWT_REFRESH_EXPIRES_IN=7d\n" +
		"TOGETHER_API_KEY=file-key\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(dotenv), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("DB_NAME", "from-env")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MongoURI != "mongodb://from-file:27017" {
		t.Errorf("got URI %q, want the .env value", cfg.MongoURI)
	}
	if cfg.MongoDatabase != "from-env" {
		t.Errorf("got database %q, want the environment to win", cfg.MongoDatabase)
	}
}
