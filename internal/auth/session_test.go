package auth_test

import (
	"testing"
	"waffle-app/internal/auth"
)

func TestCreateAndGetSession(t *testing.T) {
	store := auth.NewStore()

	token, err := store.Create("alice")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	session, ok := store.Get(token)
	if !ok {
		t.Fatal("expected session to exist")
	}
	if session.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", session.Username)
	}
}

func TestGetSession_InvalidToken(t *testing.T) {
	store := auth.NewStore()

	_, ok := store.Get("invalid-token")
	if ok {
		t.Fatal("expected no session for invalid token")
	}
}

func TestSessionTokensAreUnique(t *testing.T) {
	store := auth.NewStore()

	token1, err := store.Create("alice")
	if err != nil {
		t.Fatalf("Create token1: %v", err)
	}
	token2, err := store.Create("bob")
	if err != nil {
		t.Fatalf("Create token2: %v", err)
	}

	if token1 == token2 {
		t.Error("tokens should be unique")
	}
}
