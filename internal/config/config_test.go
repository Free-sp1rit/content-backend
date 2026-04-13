package config

import (
	"strings"
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DB_USER", "content_user")
	t.Setenv("DB_PASSWORD", "content_pass")
	t.Setenv("DB_NAME", "content_db")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
}

func assertErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("got err %q, want substring %q", err.Error(), want)
	}
}

func TestLoad(t *testing.T) {
	t.Run("uses defaults with required env only", func(t *testing.T) {
		setRequiredEnv(t)

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}

		if cfg.Server.Port != defaultServerPort {
			t.Fatalf("got server port %q, want %q", cfg.Server.Port, defaultServerPort)
		}
		if cfg.Server.ReadHeaderTimeout != defaultReadHeaderTTL {
			t.Fatalf("got read header timeout %v, want %v", cfg.Server.ReadHeaderTimeout, defaultReadHeaderTTL)
		}

		if cfg.Database.Host != defaultDBHost {
			t.Fatalf("got db host %q, want %q", cfg.Database.Host, defaultDBHost)
		}
		if cfg.Database.Port != defaultDBPort {
			t.Fatalf("got db port %q, want %q", cfg.Database.Port, defaultDBPort)
		}
		if cfg.Database.User != "content_user" {
			t.Fatalf("got db user %q, want %q", cfg.Database.User, "content_user")
		}
		if cfg.Database.Password != "content_pass" {
			t.Fatalf("got db password %q, want %q", cfg.Database.Password, "content_pass")
		}
		if cfg.Database.Name != "content_db" {
			t.Fatalf("got db name %q, want %q", cfg.Database.Name, "content_db")
		}
		if cfg.Database.SSLMode != defaultDBSSLMode {
			t.Fatalf("got db sslmode %q, want %q", cfg.Database.SSLMode, defaultDBSSLMode)
		}

		if cfg.JWT.Secret != "test-jwt-secret" {
			t.Fatalf("got jwt secret %q, want %q", cfg.JWT.Secret, "test-jwt-secret")
		}
		if cfg.JWT.Issuer != defaultJWTIssuer {
			t.Fatalf("got jwt issuer %q, want %q", cfg.JWT.Issuer, defaultJWTIssuer)
		}
		if cfg.JWT.TokenTTL != defaultJWTTokenTTL {
			t.Fatalf("got jwt token ttl %v, want %v", cfg.JWT.TokenTTL, defaultJWTTokenTTL)
		}
	})

	t.Run("uses explicit env overrides", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("PORT", "9090")
		t.Setenv("READ_HEADER_TIMEOUT", "7s")
		t.Setenv("DB_HOST", "10.0.0.2")
		t.Setenv("DB_PORT", "6543")
		t.Setenv("DB_SSLMODE", "require")
		t.Setenv("JWT_ISSUER", "custom-issuer")
		t.Setenv("JWT_TOKEN_TTL", "36h")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}

		if cfg.Server.Port != "9090" {
			t.Fatalf("got server port %q, want %q", cfg.Server.Port, "9090")
		}
		if cfg.Server.ReadHeaderTimeout != 7*time.Second {
			t.Fatalf("got read header timeout %v, want %v", cfg.Server.ReadHeaderTimeout, 7*time.Second)
		}

		if cfg.Database.Host != "10.0.0.2" {
			t.Fatalf("got db host %q, want %q", cfg.Database.Host, "10.0.0.2")
		}
		if cfg.Database.Port != "6543" {
			t.Fatalf("got db port %q, want %q", cfg.Database.Port, "6543")
		}
		if cfg.Database.SSLMode != "require" {
			t.Fatalf("got db sslmode %q, want %q", cfg.Database.SSLMode, "require")
		}

		if cfg.JWT.Issuer != "custom-issuer" {
			t.Fatalf("got jwt issuer %q, want %q", cfg.JWT.Issuer, "custom-issuer")
		}
		if cfg.JWT.TokenTTL != 36*time.Hour {
			t.Fatalf("got jwt token ttl %v, want %v", cfg.JWT.TokenTTL, 36*time.Hour)
		}
	})

	t.Run("database config is incomplete when required env missing", func(t *testing.T) {
		cases := []struct {
			name       string
			missingKey string
		}{
			{name: "missing db user", missingKey: "DB_USER"},
			{name: "missing db password", missingKey: "DB_PASSWORD"},
			{name: "missing db name", missingKey: "DB_NAME"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				setRequiredEnv(t)
				t.Setenv(tc.missingKey, "")

				_, err := Load()
				assertErrContains(t, err, "database config is incomplete")
			})
		}
	})

	t.Run("jwt secret is required", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("JWT_SECRET", "")

		_, err := Load()
		assertErrContains(t, err, "JWT_SECRET is required")
	})

	t.Run("invalid read header timeout", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("READ_HEADER_TIMEOUT", "not-a-duration")

		_, err := Load()
		assertErrContains(t, err, "parse READ_HEADER_TIMEOUT")
	})

	t.Run("invalid jwt token ttl", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("JWT_TOKEN_TTL", "not-a-duration")

		_, err := Load()
		assertErrContains(t, err, "parse JWT_TOKEN_TTL")
	})
}

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     "5432",
		User:     "content_user",
		Password: "content_pass",
		Name:     "content_db",
		SSLMode:  "disable",
	}

	got := cfg.DSN()
	want := "host=127.0.0.1 port=5432 user=content_user password=content_pass dbname=content_db sslmode=disable"

	if got != want {
		t.Fatalf("got dsn %q, want %q", got, want)
	}
}
