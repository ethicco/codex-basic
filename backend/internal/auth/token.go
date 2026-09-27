package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	AccessToken  = "access"
	RefreshToken = "refresh"
)

type Claims struct {
	Subject   string `json:"sub"`
	Email     string `json:"email"`
	SessionID string `json:"sid"`
	Type      string `json:"typ"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	ID        string `json:"jti"`
}

type TokenPair struct {
	AccessToken    string
	RefreshToken   string
	RefreshTokenID string
	ExpiresIn      int64
}

type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func NewManager(secret []byte, accessTTL, refreshTTL time.Duration) (*Manager, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes")
	}
	if accessTTL <= 0 || refreshTTL <= accessTTL {
		return nil, errors.New("invalid token TTLs")
	}
	return &Manager{secret: append([]byte(nil), secret...), accessTTL: accessTTL, refreshTTL: refreshTTL, now: time.Now}, nil
}

func (m *Manager) Issue(userID, email, sessionID string) (TokenPair, error) {
	access, _, err := m.sign(userID, email, sessionID, AccessToken, m.accessTTL)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, refreshID, err := m.sign(userID, email, sessionID, RefreshToken, m.refreshTTL)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh, RefreshTokenID: refreshID, ExpiresIn: int64(m.accessTTL.Seconds())}, nil
}

func (m *Manager) Verify(token, expectedType string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token format")
	}
	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "HS256" || header.Type != "JWT" {
		return Claims{}, errors.New("invalid token header")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, errors.New("invalid token signature")
	}
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if subtle.ConstantTimeCompare(signature, mac.Sum(nil)) != 1 {
		return Claims{}, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, errors.New("invalid token payload")
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Subject == "" || claims.Email == "" || claims.SessionID == "" || claims.ID == "" || claims.Type != expectedType || claims.ExpiresAt <= m.now().Unix() {
		return Claims{}, errors.New("invalid or expired token")
	}
	return claims, nil
}

func (m *Manager) sign(userID, email, sessionID, tokenType string, ttl time.Duration) (string, string, error) {
	id, err := NewID()
	if err != nil {
		return "", "", err
	}
	now := m.now().UTC()
	claims := Claims{Subject: userID, Email: email, SessionID: sessionID, Type: tokenType, IssuedAt: now.Unix(), ExpiresAt: now.Add(ttl).Unix(), ID: id}
	header, err := json.Marshal(struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}{Algorithm: "HS256", Type: "JWT"})
	if err != nil {
		return "", "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", "", err
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(input))
	return input + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), id, nil
}

func NewID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate identifier: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func Digest(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}
