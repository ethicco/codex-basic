package monitor

import "testing"

func TestValidateTargetURL(t *testing.T) {
	for _, target := range []string{"https://example.com/path", "http://example.com"} {
		if _, err := ValidateTargetURL(target); err != nil {
			t.Fatalf("ValidateTargetURL(%q) error = %v", target, err)
		}
	}
	for _, target := range []string{"example.com", "ftp://example.com", "http://127.0.0.1", "http://[::1]", "http://192.168.1.10"} {
		if _, err := ValidateTargetURL(target); err == nil {
			t.Fatalf("ValidateTargetURL(%q) accepted a disallowed target", target)
		}
	}
}
