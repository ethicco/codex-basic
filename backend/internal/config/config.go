// Package config loads runtime settings for the backend service.
package config

import (
	"fmt"
	"os"
	"time"
)

const (
	defaultAddress         = ":8080"
	defaultAccessTokenTTL  = 15 * time.Minute
	defaultRefreshTokenTTL = 30 * 24 * time.Hour
)

type Config struct {
	Address         string
	DatabaseURL     string
	JWTSecret       []byte
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	AvatarDir       string
}

func Load() (Config, error) {
	accessTTL, err := durationFromEnv("AUTH_ACCESS_TOKEN_TTL", defaultAccessTokenTTL)
	if err != nil {
		return Config{}, err
	}
	refreshTTL, err := durationFromEnv("AUTH_REFRESH_TOKEN_TTL", defaultRefreshTokenTTL)
	if err != nil {
		return Config{}, err
	}
	if accessTTL <= 0 || refreshTTL <= accessTTL {
		return Config{}, fmt.Errorf("AUTH_REFRESH_TOKEN_TTL must be longer than AUTH_ACCESS_TOKEN_TTL")
	}
	secret := os.Getenv("AUTH_JWT_SECRET")
	if len(secret) < 32 {
		return Config{}, fmt.Errorf("AUTH_JWT_SECRET must be at least 32 bytes")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	address := os.Getenv("SERVER_ADDR")
	if address == "" {
		address = defaultAddress
	}
	avatarDir := os.Getenv("AVATAR_DIR")
	if avatarDir == "" {
		avatarDir = "data/avatars"
	}
	return Config{Address: address, DatabaseURL: databaseURL, JWTSecret: []byte(secret), AccessTokenTTL: accessTTL, RefreshTokenTTL: refreshTTL, AvatarDir: avatarDir}, nil
}

func durationFromEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return duration, nil
}
