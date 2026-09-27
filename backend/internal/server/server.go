// Package server exposes the HTTP authentication API.
package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"uptime-backend/internal/auth"
	"uptime-backend/internal/config"
	"uptime-backend/internal/user"
)

const (
	maxRequestBodyBytes = 1 << 20
	maxAvatarBytes      = 5 << 20
)

type api struct {
	repository      user.Repository
	tokens          *auth.Manager
	refreshTokenTTL time.Duration
	now             func() time.Time
	avatarDir       string
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
type profileRequest struct {
	Name string `json:"name"`
}
type monitorRequest struct {
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
}
type publicUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
type tokenResponse struct {
	User         publicUser `json:"user"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	ExpiresIn    int64      `json:"expires_in"`
}
type publicProfile struct {
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type publicMonitor struct {
	ID              string    `json:"id"`
	URL             string    `json:"url"`
	IntervalSeconds int       `json:"interval_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}

func New(cfg config.Config, repository user.Repository) (http.Handler, error) {
	if repository == nil {
		return nil, errors.New("user repository is required")
	}
	tokens, err := auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		return nil, err
	}
	if cfg.AvatarDir == "" {
		return nil, errors.New("avatar directory is required")
	}
	if err := os.MkdirAll(cfg.AvatarDir, 0o750); err != nil {
		return nil, fmt.Errorf("create avatar directory: %w", err)
	}
	api := &api{repository: repository, tokens: tokens, refreshTokenTTL: cfg.RefreshTokenTTL, now: time.Now, avatarDir: cfg.AvatarDir}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", api.register)
	mux.HandleFunc("POST /api/auth/login", api.login)
	mux.HandleFunc("POST /api/auth/refresh", api.refresh)
	mux.HandleFunc("POST /api/auth/logout", api.logout)
	mux.HandleFunc("GET /api/auth/me", api.me)
	mux.HandleFunc("GET /api/profile", api.getProfile)
	mux.HandleFunc("POST /api/profile", api.createProfile)
	mux.HandleFunc("PATCH /api/profile", api.updateProfile)
	mux.HandleFunc("GET /api/profile/avatar", api.getAvatar)
	mux.HandleFunc("POST /api/profile/avatar", api.uploadAvatar)
	mux.HandleFunc("GET /api/monitors", api.listMonitors)
	mux.HandleFunc("POST /api/monitors", api.createMonitor)
	return mux, nil
}

func (a *api) listMonitors(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticatedUser(w, r)
	if !ok {
		return
	}
	monitors, err := a.repository.ListMonitorsByUserID(r.Context(), account.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load monitors")
		return
	}
	public := make([]publicMonitor, len(monitors))
	for i, monitor := range monitors {
		public[i] = publicMonitorFrom(monitor)
	}
	writeJSON(w, http.StatusOK, map[string][]publicMonitor{"monitors": public})
}

func (a *api) createMonitor(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticatedUser(w, r)
	if !ok {
		return
	}
	request, ok := decodeMonitor(w, r)
	if !ok {
		return
	}
	id, err := randomID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create monitor")
		return
	}
	monitor := user.Monitor{ID: id, UserID: account.ID, TargetURL: request.URL, IntervalSeconds: request.IntervalSeconds, CreatedAt: a.now().UTC()}
	if err := a.repository.CreateMonitor(r.Context(), monitor); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create monitor")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]publicMonitor{"monitor": publicMonitorFrom(monitor)})
}

func (a *api) register(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	email, err := normalizeEmail(request.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_email", err.Error())
		return
	}
	passwordHash, err := auth.HashPassword(request.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_password", err.Error())
		return
	}
	accountID, err := randomID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create account")
		return
	}
	account := user.User{ID: accountID, Email: email, PasswordHash: passwordHash, CreatedAt: a.now().UTC()}
	pair, session, err := a.newSession(account)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not issue tokens")
		return
	}
	if err := a.repository.CreateUserWithSession(r.Context(), account, session); err != nil {
		if errors.Is(err, user.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "email_already_registered", "an account with this email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create account")
		return
	}
	a.writeTokenResponse(w, http.StatusCreated, account, pair)
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	email, err := normalizeEmail(request.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
		return
	}
	account, found, err := a.repository.FindByEmail(r.Context(), email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not sign in")
		return
	}
	if !found || !auth.VerifyPassword(account.PasswordHash, request.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
		return
	}
	pair, session, err := a.newSession(account)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not issue tokens")
		return
	}
	if err := a.repository.CreateSession(r.Context(), session); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not sign in")
		return
	}
	a.writeTokenResponse(w, http.StatusOK, account, pair)
}

func (a *api) refresh(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeRefresh(w, r)
	if !ok {
		return
	}
	claims, err := a.tokens.Verify(request.RefreshToken, auth.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh token is invalid or expired")
		return
	}
	account, found, err := a.repository.FindByEmail(r.Context(), claims.Email)
	if err != nil || !found || account.ID != claims.Subject {
		writeError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh token is invalid or expired")
		return
	}
	pair, err := a.tokens.Issue(claims.Subject, claims.Email, claims.SessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not issue tokens")
		return
	}
	session := user.Session{ID: claims.SessionID, UserID: claims.Subject, RefreshJTIHash: auth.Digest(pair.RefreshTokenID), ExpiresAt: a.now().UTC().Add(a.refreshTokenTTL)}
	if err := a.repository.RotateSession(r.Context(), session, auth.Digest(claims.ID)); err != nil {
		if errors.Is(err, user.ErrSessionInvalid) {
			writeError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh token is invalid or expired")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not refresh token")
		return
	}
	a.writeTokenResponse(w, http.StatusOK, account, pair)
}

func (a *api) logout(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeRefresh(w, r)
	if !ok {
		return
	}
	claims, err := a.tokens.Verify(request.RefreshToken, auth.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh token is invalid or expired")
		return
	}
	if err := a.repository.RevokeSession(r.Context(), claims.SessionID, claims.Subject, auth.Digest(claims.ID)); err != nil {
		if errors.Is(err, user.ErrSessionInvalid) {
			writeError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh token is invalid or expired")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not log out")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) me(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticatedUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]publicUser{
		"user": {ID: account.ID, Email: account.Email, CreatedAt: account.CreatedAt},
	})
}

func (a *api) getProfile(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticatedUser(w, r)
	if !ok {
		return
	}
	profile, found, err := a.repository.FindProfileByUserID(r.Context(), account.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load profile")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "profile_not_found", "profile has not been created")
		return
	}
	writeJSON(w, http.StatusOK, map[string]publicProfile{"profile": publicProfileFrom(profile)})
}

func (a *api) createProfile(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticatedUser(w, r)
	if !ok {
		return
	}
	request, ok := decodeProfile(w, r)
	if !ok {
		return
	}
	now := a.now().UTC()
	profile := user.Profile{UserID: account.ID, Name: request.Name, CreatedAt: now, UpdatedAt: now}
	if err := a.repository.CreateProfile(r.Context(), profile); err != nil {
		if errors.Is(err, user.ErrProfileExists) {
			writeError(w, http.StatusConflict, "profile_already_exists", "profile already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create profile")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]publicProfile{"profile": publicProfileFrom(profile)})
}

func (a *api) updateProfile(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticatedUser(w, r)
	if !ok {
		return
	}
	request, ok := decodeProfile(w, r)
	if !ok {
		return
	}
	profile, err := a.repository.UpdateProfileName(r.Context(), account.ID, request.Name)
	if errors.Is(err, user.ErrProfileMissing) {
		writeError(w, http.StatusNotFound, "profile_not_found", "profile has not been created")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update profile")
		return
	}
	writeJSON(w, http.StatusOK, map[string]publicProfile{"profile": publicProfileFrom(profile)})
}

func (a *api) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticatedUser(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarBytes)
	if err := r.ParseMultipartForm(maxAvatarBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_avatar", "avatar image must be smaller than 5 MB")
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_avatar", "avatar image is required")
		return
	}
	defer file.Close()

	contents, err := io.ReadAll(io.LimitReader(file, maxAvatarBytes+1))
	if err != nil || len(contents) > maxAvatarBytes {
		writeError(w, http.StatusBadRequest, "invalid_avatar", "avatar image must be smaller than 5 MB")
		return
	}
	contentType := http.DetectContentType(contents)
	extension, ok := avatarExtension(contentType)
	if !ok {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_avatar", "avatar must be a JPEG, PNG, GIF, or WebP image")
		return
	}
	if header.Size == 0 {
		writeError(w, http.StatusBadRequest, "invalid_avatar", "avatar image is empty")
		return
	}
	profile, found, err := a.repository.FindProfileByUserID(r.Context(), account.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load profile")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "profile_not_found", "profile has not been created")
		return
	}
	fileID, err := randomID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not save avatar")
		return
	}
	filename := fileID + extension
	path := filepath.Join(a.avatarDir, filename)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not save avatar")
		return
	}
	updated, err := a.repository.UpdateProfileAvatar(r.Context(), account.ID, filename)
	if err != nil {
		_ = os.Remove(path)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not save avatar")
		return
	}
	if profile.AvatarFile != "" {
		_ = os.Remove(filepath.Join(a.avatarDir, filepath.Base(profile.AvatarFile)))
	}
	writeJSON(w, http.StatusOK, map[string]publicProfile{"profile": publicProfileFrom(updated)})
}

func (a *api) getAvatar(w http.ResponseWriter, r *http.Request) {
	account, ok := a.authenticatedUser(w, r)
	if !ok {
		return
	}
	profile, found, err := a.repository.FindProfileByUserID(r.Context(), account.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load profile")
		return
	}
	if !found || profile.AvatarFile == "" {
		writeError(w, http.StatusNotFound, "avatar_not_found", "avatar has not been uploaded")
		return
	}
	path := filepath.Join(a.avatarDir, filepath.Base(profile.AvatarFile))
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "avatar_not_found", "avatar has not been uploaded")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load avatar")
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(contents))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(contents)
}

func avatarExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func (a *api) authenticatedUser(w http.ResponseWriter, r *http.Request) (user.User, bool) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || parts[0] != "Bearer" {
		writeError(w, http.StatusUnauthorized, "invalid_access_token", "access token is invalid or expired")
		return user.User{}, false
	}
	claims, err := a.tokens.Verify(parts[1], auth.AccessToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_access_token", "access token is invalid or expired")
		return user.User{}, false
	}
	account, found, err := a.repository.FindByEmail(r.Context(), claims.Email)
	if err != nil || !found || account.ID != claims.Subject {
		writeError(w, http.StatusUnauthorized, "invalid_access_token", "access token is invalid or expired")
		return user.User{}, false
	}
	return account, true
}

func decodeProfile(w http.ResponseWriter, r *http.Request) (profileRequest, bool) {
	var request profileRequest
	if !decodeJSON(w, r, &request) {
		return profileRequest{}, false
	}
	request.Name = strings.TrimSpace(request.Name)
	if length := len([]rune(request.Name)); length < 1 || length > 100 {
		writeError(w, http.StatusBadRequest, "invalid_name", "name must be between 1 and 100 characters")
		return profileRequest{}, false
	}
	return request, true
}

func decodeMonitor(w http.ResponseWriter, r *http.Request) (monitorRequest, bool) {
	var request monitorRequest
	if !decodeJSON(w, r, &request) {
		return monitorRequest{}, false
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(request.URL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		writeError(w, http.StatusBadRequest, "invalid_url", "url must be an absolute HTTP or HTTPS URL")
		return monitorRequest{}, false
	}
	if request.IntervalSeconds != 5 && request.IntervalSeconds != 60 && request.IntervalSeconds != 3600 {
		writeError(w, http.StatusBadRequest, "invalid_interval", "interval_seconds must be 5, 60, or 3600")
		return monitorRequest{}, false
	}
	request.URL = parsed.String()
	return request, true
}

func publicProfileFrom(profile user.Profile) publicProfile {
	avatarURL := ""
	if profile.AvatarFile != "" {
		avatarURL = "/api/profile/avatar"
	}
	return publicProfile{Name: profile.Name, AvatarURL: avatarURL, CreatedAt: profile.CreatedAt, UpdatedAt: profile.UpdatedAt}
}

func publicMonitorFrom(monitor user.Monitor) publicMonitor {
	return publicMonitor{ID: monitor.ID, URL: monitor.TargetURL, IntervalSeconds: monitor.IntervalSeconds, CreatedAt: monitor.CreatedAt}
}

func (a *api) newSession(account user.User) (auth.TokenPair, user.Session, error) {
	sessionID, err := randomID()
	if err != nil {
		return auth.TokenPair{}, user.Session{}, err
	}
	pair, err := a.tokens.Issue(account.ID, account.Email, sessionID)
	if err != nil {
		return auth.TokenPair{}, user.Session{}, err
	}
	now := a.now().UTC()
	return pair, user.Session{ID: sessionID, UserID: account.ID, RefreshJTIHash: auth.Digest(pair.RefreshTokenID), ExpiresAt: now.Add(a.refreshTokenTTL), CreatedAt: now}, nil
}

func (a *api) writeTokenResponse(w http.ResponseWriter, status int, account user.User, pair auth.TokenPair) {
	writeJSON(w, status, tokenResponse{User: publicUser{ID: account.ID, Email: account.Email, CreatedAt: account.CreatedAt}, AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, TokenType: "Bearer", ExpiresIn: pair.ExpiresIn})
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (credentialsRequest, bool) {
	var request credentialsRequest
	if !decodeJSON(w, r, &request) {
		return credentialsRequest{}, false
	}
	if strings.TrimSpace(request.Email) == "" || request.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "email and password are required")
		return credentialsRequest{}, false
	}
	return request, true
}

func decodeRefresh(w http.ResponseWriter, r *http.Request) (refreshRequest, bool) {
	var request refreshRequest
	if !decodeJSON(w, r, &request) {
		return refreshRequest{}, false
	}
	if request.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return refreshRequest{}, false
	}
	return request, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body must contain one JSON object")
		return false
	}
	return true
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if len(email) > 254 {
		return "", errors.New("email is too long")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", errors.New("email must be a valid address")
	}
	return email, nil
}

func randomID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
