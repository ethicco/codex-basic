package auth

import (
	"testing"
	"time"
)

func TestManagerIssuesTypedSessionTokens(t *testing.T) {
	manager, err := NewManager([]byte("01234567890123456789012345678901"), time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	pair, err := manager.Issue("user-1", "person@example.com", "session-1")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	claims, err := manager.Verify(pair.AccessToken, AccessToken)
	if err != nil {
		t.Fatalf("Verify(access) error = %v", err)
	}
	if claims.Subject != "user-1" || claims.Email != "person@example.com" || claims.SessionID != "session-1" {
		t.Fatalf("claims = %#v", claims)
	}
	if _, err := manager.Verify(pair.AccessToken, RefreshToken); err == nil {
		t.Fatal("Verify() accepted access token as refresh token")
	}
	if pair.RefreshTokenID == "" || len(Digest(pair.RefreshTokenID)) != 32 {
		t.Fatal("refresh identifier was not generated")
	}
}
