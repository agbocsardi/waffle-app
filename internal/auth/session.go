package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const sessionCookieName = "waffle_session"

// Session holds the authenticated user's data for the duration of a request.
type Session struct {
	Username string
}

// Store is an in-memory session store.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]sessionEntry
}

type sessionEntry struct {
	username  string
	createdAt time.Time
}

func NewStore() *Store {
	return &Store{sessions: make(map[string]sessionEntry)}
}

// Create generates a new session token and stores it.
func (s *Store) Create(username string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	s.mu.Lock()
	s.sessions[token] = sessionEntry{username: username, createdAt: time.Now()}
	s.mu.Unlock()
	slog.Info("session created", "username", username)
	return token, nil
}

// Get retrieves the session associated with the token.
func (s *Store) Get(token string) (*Session, bool) {
	s.mu.RLock()
	entry, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return &Session{Username: entry.username}, true
}

// SetCookie writes the session cookie to the response.
func SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// FromRequest extracts the session token from the request cookie.
func FromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}
	token := strings.TrimSpace(cookie.Value)
	if token == "" {
		return "", false
	}
	return token, true
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
