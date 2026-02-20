package conversations

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"waffle-app/internal/auth"
	"waffle-app/internal/storage"
)

type Handler struct {
	DB       *storage.DB
	Sessions *auth.Store
}

func NewHandler(db *storage.DB, sessions *auth.Store) *Handler {
	return &Handler{DB: db, Sessions: sessions}
}

// POST /api/conversations
// Body: { "name": "College Friends" }
// Response: { "id": "...", "invite_code": "..." }
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireSession(w, r)
	if !ok {
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "invalid body: 'name' is required", http.StatusBadRequest)
		return
	}

	id, err := generateID()
	if err != nil {
		slog.Error("failed to generate conversation id", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	inviteCode, err := generateInviteCode()
	if err != nil {
		slog.Error("failed to generate invite code", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.DB.CreateConversation(id, inviteCode, body.Name); err != nil {
		slog.Error("failed to create conversation", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Creator automatically joins the conversation
	if err := h.DB.AddMember(id, session.Username); err != nil {
		slog.Error("failed to add creator as member", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Info("conversation created", "id", id, "name", body.Name, "creator", session.Username)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":          id,
		"invite_code": inviteCode,
		"name":        body.Name,
	})
}

// GET /api/conversations
// Response: [{ "id": "...", "invite_code": "...", "name": "..." }, ...]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireSession(w, r)
	if !ok {
		return
	}

	conversations, err := h.DB.GetConversationsByUsername(session.Username)
	if err != nil {
		slog.Error("failed to list conversations", "error", err, "username", session.Username)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Debug("listed conversations", "username", session.Username, "count", len(conversations))

	type response struct {
		ID         string `json:"id"`
		InviteCode string `json:"invite_code"`
		Name       string `json:"name"`
	}
	result := make([]response, 0, len(conversations))
	for _, c := range conversations {
		result = append(result, response{ID: c.ID, InviteCode: c.InviteCode, Name: c.Name})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/conversations/join
// Body: { "invite_code": "...", "username": "..." }
func (h *Handler) Join(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InviteCode string `json:"invite_code"`
		Username   string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.InviteCode == "" || body.Username == "" {
		http.Error(w, "invalid body: 'invite_code' and 'username' are required", http.StatusBadRequest)
		return
	}

	conversation, err := h.DB.GetConversationByInviteCode(body.InviteCode)
	if err != nil {
		slog.Error("failed to look up invite code", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if conversation == nil {
		slog.Warn("invalid invite code used", "invite_code", body.InviteCode, "username", body.Username)
		http.Error(w, "invalid invite code", http.StatusUnauthorized)
		return
	}

	if err := h.DB.AddMember(conversation.ID, body.Username); err != nil {
		slog.Error("failed to add member", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := h.Sessions.Create(body.Username)
	if err != nil {
		slog.Error("failed to create session", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	auth.SetCookie(w, token)
	slog.Info("user joined conversation", "username", body.Username, "conversation_id", conversation.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"conversation_id": conversation.ID,
		"username":        body.Username,
	})
}

func (h *Handler) requireSession(w http.ResponseWriter, r *http.Request) (*auth.Session, bool) {
	token, ok := auth.FromRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	session, ok := h.Sessions.Get(token)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return session, true
}

func generateID() (string, error) {
	return generateHex(16)
}

func generateInviteCode() (string, error) {
	return generateHex(6)
}

func generateHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate hex: %w", err)
	}
	return hex.EncodeToString(b), nil
}
