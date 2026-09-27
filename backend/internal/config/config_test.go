package config

import "testing"

func TestLoadReadsAuthenticationSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("AUTH_JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("AUTH_ACCESS_TOKEN_TTL", "10m")
	t.Setenv("AUTH_REFRESH_TOKEN_TTL", "48h")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AccessTokenTTL.String() != "10m0s" || cfg.RefreshTokenTTL.String() != "48h0m0s" {
		t.Fatalf("config = %#v", cfg)
	}
}

func TestLoadRequiresDatabaseURLAndSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("AUTH_JWT_SECRET", "too-short")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted missing database URL and short secret")
	}
}
