package server

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"uptime-backend/internal/config"
	"uptime-backend/internal/user"
)

type authPayload struct {
	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type memoryRepository struct {
	mu       sync.Mutex
	users    map[string]user.User
	sessions map[string]user.Session
	revoked  map[string]bool
	profiles map[string]user.Profile
	monitors map[string][]user.Monitor
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{users: map[string]user.User{}, sessions: map[string]user.Session{}, revoked: map[string]bool{}, profiles: map[string]user.Profile{}, monitors: map[string][]user.Monitor{}}
}

func (r *memoryRepository) CreateUserWithSession(_ context.Context, account user.User, session user.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.users[account.Email]; exists {
		return user.ErrEmailTaken
	}
	r.users[account.Email] = account
	r.sessions[session.ID] = session
	return nil
}
func (r *memoryRepository) FindByEmail(_ context.Context, email string) (user.User, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	account, found := r.users[email]
	return account, found, nil
}
func (r *memoryRepository) CreateSession(_ context.Context, session user.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
	return nil
}
func (r *memoryRepository) RotateSession(_ context.Context, session user.Session, previousHash []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, found := r.sessions[session.ID]
	if !found || r.revoked[session.ID] || current.UserID != session.UserID || !bytes.Equal(current.RefreshJTIHash, previousHash) {
		r.revoked[session.ID] = true
		return user.ErrSessionInvalid
	}
	r.sessions[session.ID] = session
	return nil
}
func (r *memoryRepository) RevokeSession(_ context.Context, sessionID, userID string, refreshHash []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, found := r.sessions[sessionID]
	if !found || r.revoked[sessionID] || current.UserID != userID || !bytes.Equal(current.RefreshJTIHash, refreshHash) {
		return user.ErrSessionInvalid
	}
	r.revoked[sessionID] = true
	return nil
}
func (r *memoryRepository) CreateProfile(_ context.Context, profile user.Profile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.profiles[profile.UserID]; exists {
		return user.ErrProfileExists
	}
	r.profiles[profile.UserID] = profile
	return nil
}
func (r *memoryRepository) FindProfileByUserID(_ context.Context, userID string) (user.Profile, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	profile, found := r.profiles[userID]
	return profile, found, nil
}
func (r *memoryRepository) UpdateProfileName(_ context.Context, userID, name string) (user.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	profile, found := r.profiles[userID]
	if !found {
		return user.Profile{}, user.ErrProfileMissing
	}
	profile.Name = name
	profile.UpdatedAt = time.Now().UTC()
	r.profiles[userID] = profile
	return profile, nil
}
func (r *memoryRepository) UpdateProfileAvatar(_ context.Context, userID, avatarFile string) (user.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	profile, found := r.profiles[userID]
	if !found {
		return user.Profile{}, user.ErrProfileMissing
	}
	profile.AvatarFile = avatarFile
	profile.UpdatedAt = time.Now().UTC()
	r.profiles[userID] = profile
	return profile, nil
}
func (r *memoryRepository) CreateMonitor(_ context.Context, monitor user.Monitor) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.monitors[monitor.UserID] = append([]user.Monitor{monitor}, r.monitors[monitor.UserID]...)
	return nil
}
func (r *memoryRepository) ListMonitorsByUserID(_ context.Context, userID string) ([]user.Monitor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]user.Monitor(nil), r.monitors[userID]...), nil
}

func TestAuthenticationFlow(t *testing.T) {
	handler := newTestHandler(t)
	registered := callJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{"email": "Person@Example.com", "password": "a sufficiently long password"})
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", registered.Code, registered.Body.String())
	}
	var registration authPayload
	decodeResponse(t, registered, &registration)
	if registration.User.Email != "person@example.com" || registration.AccessToken == "" || registration.RefreshToken == "" || registration.TokenType != "Bearer" {
		t.Fatalf("registration payload = %#v", registration)
	}

	duplicate := callJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{"email": "person@example.com", "password": "a different sufficiently long password"})
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d", duplicate.Code)
	}
	badLogin := callJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]string{"email": "person@example.com", "password": "a wrong sufficiently long password"})
	if badLogin.Code != http.StatusUnauthorized {
		t.Fatalf("bad login status = %d", badLogin.Code)
	}
	login := callJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]string{"email": "person@example.com", "password": "a sufficiently long password"})
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d", login.Code)
	}
	var loggedIn authPayload
	decodeResponse(t, login, &loggedIn)

	refreshed := callJSON(t, handler, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": loggedIn.RefreshToken})
	if refreshed.Code != http.StatusOK {
		t.Fatalf("refresh status = %d", refreshed.Code)
	}
	var refreshedPayload authPayload
	decodeResponse(t, refreshed, &refreshedPayload)
	if refreshedPayload.AccessToken == loggedIn.AccessToken || refreshedPayload.RefreshToken == loggedIn.RefreshToken {
		t.Fatal("refresh did not issue a new pair")
	}
	replay := callJSON(t, handler, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": loggedIn.RefreshToken})
	if replay.Code != http.StatusUnauthorized {
		t.Fatalf("replay status = %d", replay.Code)
	}
	invalidated := callJSON(t, handler, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": refreshedPayload.RefreshToken})
	if invalidated.Code != http.StatusUnauthorized {
		t.Fatalf("replay did not revoke the session, got %d", invalidated.Code)
	}
}

func TestLogoutAndRequestValidation(t *testing.T) {
	handler := newTestHandler(t)
	registered := callJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{"email": "person@example.com", "password": "a sufficiently long password"})
	var payload authPayload
	decodeResponse(t, registered, &payload)
	wrongType := callJSON(t, handler, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": payload.AccessToken})
	if wrongType.Code != http.StatusUnauthorized {
		t.Fatalf("access refresh status = %d", wrongType.Code)
	}
	logout := callJSON(t, handler, http.MethodPost, "/api/auth/logout", map[string]string{"refresh_token": payload.RefreshToken})
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d", logout.Code)
	}
	refreshed := callJSON(t, handler, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": payload.RefreshToken})
	if refreshed.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout status = %d", refreshed.Code)
	}
	invalidContentType := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString("{}"))
	invalidContentType.Header.Set("Content-Type", "text/plain")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, invalidContentType)
	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("content type status = %d", recorder.Code)
	}
}

func TestMeReturnsAuthenticatedUser(t *testing.T) {
	handler := newTestHandler(t)
	registered := callJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{"email": "person@example.com", "password": "a sufficiently long password"})
	var payload authPayload
	decodeResponse(t, registered, &payload)

	request := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+payload.AccessToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("me status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	decodeResponse(t, response, &body)
	if body.User.Email != "person@example.com" {
		t.Fatalf("me response = %#v", body)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+payload.RefreshToken)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("me with refresh token status = %d", response.Code)
	}
}

func TestProfileLifecycle(t *testing.T) {
	handler := newTestHandler(t)
	registered := callJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{"email": "person@example.com", "password": "a sufficiently long password"})
	var payload authPayload
	decodeResponse(t, registered, &payload)

	created := callJSONWithAccessToken(t, handler, http.MethodPost, "/api/profile", payload.AccessToken, map[string]string{"name": "Анна"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create profile status = %d, body = %s", created.Code, created.Body.String())
	}
	loaded := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	request.Header.Set("Authorization", "Bearer "+payload.AccessToken)
	handler.ServeHTTP(loaded, request)
	if loaded.Code != http.StatusOK {
		t.Fatalf("get profile status = %d", loaded.Code)
	}
	updated := callJSONWithAccessToken(t, handler, http.MethodPatch, "/api/profile", payload.AccessToken, map[string]string{"name": "Анна Иванова"})
	if updated.Code != http.StatusOK {
		t.Fatalf("update profile status = %d, body = %s", updated.Code, updated.Body.String())
	}
	var body struct {
		Profile struct {
			Name string `json:"name"`
		} `json:"profile"`
	}
	decodeResponse(t, updated, &body)
	if body.Profile.Name != "Анна Иванова" {
		t.Fatalf("updated profile = %#v", body)
	}
}

func TestMonitorLifecycle(t *testing.T) {
	handler := newTestHandler(t)
	registered := callJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{"email": "person@example.com", "password": "a sufficiently long password"})
	var payload authPayload
	decodeResponse(t, registered, &payload)

	created := callJSONWithAccessToken(t, handler, http.MethodPost, "/api/monitors", payload.AccessToken, map[string]any{"url": "https://example.com/status", "interval_seconds": 60})
	if created.Code != http.StatusCreated {
		t.Fatalf("create monitor status = %d, body = %s", created.Code, created.Body.String())
	}
	var createdBody struct {
		Monitor publicMonitor `json:"monitor"`
	}
	decodeResponse(t, created, &createdBody)
	if createdBody.Monitor.ID == "" || createdBody.Monitor.URL != "https://example.com/status" || createdBody.Monitor.IntervalSeconds != 60 {
		t.Fatalf("created monitor = %#v", createdBody.Monitor)
	}

	listed := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/monitors", nil)
	request.Header.Set("Authorization", "Bearer "+payload.AccessToken)
	handler.ServeHTTP(listed, request)
	if listed.Code != http.StatusOK {
		t.Fatalf("list monitors status = %d, body = %s", listed.Code, listed.Body.String())
	}
	var listedBody struct {
		Monitors []publicMonitor `json:"monitors"`
	}
	decodeResponse(t, listed, &listedBody)
	if len(listedBody.Monitors) != 1 || listedBody.Monitors[0].ID != createdBody.Monitor.ID {
		t.Fatalf("listed monitors = %#v", listedBody.Monitors)
	}
}

func TestMonitorValidation(t *testing.T) {
	handler := newTestHandler(t)
	registered := callJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{"email": "person@example.com", "password": "a sufficiently long password"})
	var payload authPayload
	decodeResponse(t, registered, &payload)
	for _, request := range []map[string]any{
		{"url": "example.com", "interval_seconds": 60},
		{"url": "ftp://example.com", "interval_seconds": 60},
		{"url": "https://example.com", "interval_seconds": 30},
	} {
		response := callJSONWithAccessToken(t, handler, http.MethodPost, "/api/monitors", payload.AccessToken, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid monitor status = %d, body = %s", response.Code, response.Body.String())
		}
	}
}

func TestAvatarUploadAndRetrieval(t *testing.T) {
	handler := newTestHandler(t)
	registered := callJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{"email": "person@example.com", "password": "a sufficiently long password"})
	var payload authPayload
	decodeResponse(t, registered, &payload)
	created := callJSONWithAccessToken(t, handler, http.MethodPost, "/api/profile", payload.AccessToken, map[string]string{"name": "Анна"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create profile status = %d", created.Code)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	// A valid 1×1 PNG.
	image := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d, 0x49, 0x48, 0x44, 0x52, 0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0, 0x1f, 0x15, 0xc4, 0x89}
	if _, err := part.Write(image); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	uploadRequest := httptest.NewRequest(http.MethodPost, "/api/profile/avatar", &body)
	uploadRequest.Header.Set("Authorization", "Bearer "+payload.AccessToken)
	uploadRequest.Header.Set("Content-Type", writer.FormDataContentType())
	uploaded := httptest.NewRecorder()
	handler.ServeHTTP(uploaded, uploadRequest)
	if uploaded.Code != http.StatusOK {
		t.Fatalf("upload avatar status = %d, body = %s", uploaded.Code, uploaded.Body.String())
	}
	var uploadBody struct {
		Profile struct {
			AvatarURL string `json:"avatar_url"`
		} `json:"profile"`
	}
	decodeResponse(t, uploaded, &uploadBody)
	if uploadBody.Profile.AvatarURL != "/api/profile/avatar" {
		t.Fatalf("avatar URL = %q", uploadBody.Profile.AvatarURL)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/profile/avatar", nil)
	getRequest.Header.Set("Authorization", "Bearer "+payload.AccessToken)
	avatar := httptest.NewRecorder()
	handler.ServeHTTP(avatar, getRequest)
	if avatar.Code != http.StatusOK || avatar.Header().Get("Content-Type") != "image/png" || !bytes.Equal(avatar.Body.Bytes(), image) {
		t.Fatalf("retrieved avatar status = %d, type = %q, body = %x", avatar.Code, avatar.Header().Get("Content-Type"), avatar.Body.Bytes())
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	handler, err := New(config.Config{JWTSecret: []byte("01234567890123456789012345678901"), AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour, AvatarDir: t.TempDir()}, newMemoryRepository())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return handler
}
func callJSON(t *testing.T, handler http.Handler, method, path string, request any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	httpRequest := httptest.NewRequest(method, path, bytes.NewReader(body))
	httpRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, httpRequest)
	return recorder
}
func callJSONWithAccessToken(t *testing.T, handler http.Handler, method, path, accessToken string, request any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	httpRequest := httptest.NewRequest(method, path, bytes.NewReader(body))
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+accessToken)
	handler.ServeHTTP(recorder, httpRequest)
	return recorder
}
func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatal(err)
	}
}
