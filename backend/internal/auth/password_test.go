package auth

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	password := "a sufficiently secure password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !VerifyPassword(hash, password) {
		t.Fatal("VerifyPassword() rejected correct password")
	}
	if VerifyPassword(hash, "a different sufficiently secure password") {
		t.Fatal("VerifyPassword() accepted wrong password")
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("too-short"); err == nil {
		t.Fatal("ValidatePassword() accepted a short password")
	}
	if err := ValidatePassword("valid password"); err != nil {
		t.Fatalf("ValidatePassword() error = %v", err)
	}
}
