// Package monitor defines monitor persistence and target validation.
package monitor

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

const (
	MaxPerUser      = 100
	DefaultPageSize = 50
	MaxPageSize     = 100
)

var ErrLimitReached = errors.New("monitor limit reached")

type Monitor struct {
	ID              string
	UserID          string
	TargetURL       string
	IntervalSeconds int
	CreatedAt       time.Time
}

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

type Repository interface {
	CreateMonitor(ctx context.Context, monitor Monitor) error
	ListMonitorsByUserID(ctx context.Context, userID string, limit int, cursor *Cursor) ([]Monitor, error)
}

// ValidateTargetURL accepts only canonical HTTP(S) URLs. It rejects literal
// loopback, private, link-local, multicast, and unspecified IP targets. A
// monitor executor must also resolve and validate hostnames immediately before
// every request, including after each redirect, to prevent DNS rebinding.
func ValidateTargetURL(value string) (string, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return "", errors.New("url must be an absolute HTTP or HTTPS URL")
	}
	if address := net.ParseIP(parsed.Hostname()); address != nil && !isPublicIP(address) {
		return "", errors.New("url must not target a private network address")
	}
	return parsed.String(), nil
}

func isPublicIP(address net.IP) bool {
	return !address.IsLoopback() && !address.IsPrivate() && !address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast() && !address.IsMulticast() && !address.IsUnspecified()
}
