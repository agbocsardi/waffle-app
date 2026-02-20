package storage_test

import (
	"os"
	"testing"
	"waffle-app/internal/storage"
)

func newTestDB(t *testing.T) *storage.DB {
	t.Helper()
	f, err := os.CreateTemp("", "waffle_test_*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	db, err := storage.New(f.Name())
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCreateAndGetConversation(t *testing.T) {
	db := newTestDB(t)

	err := db.CreateConversation("conv-1", "invite-abc", "Test Group")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	conv, err := db.GetConversationByInviteCode("invite-abc")
	if err != nil {
		t.Fatalf("GetConversationByInviteCode: %v", err)
	}
	if conv == nil {
		t.Fatal("expected conversation, got nil")
	}
	if conv.ID != "conv-1" {
		t.Errorf("expected id 'conv-1', got %q", conv.ID)
	}
	if conv.Name != "Test Group" {
		t.Errorf("expected name 'Test Group', got %q", conv.Name)
	}
}

func TestGetConversationByInviteCode_NotFound(t *testing.T) {
	db := newTestDB(t)

	conv, err := db.GetConversationByInviteCode("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conv != nil {
		t.Errorf("expected nil, got %+v", conv)
	}
}

func TestDuplicateInviteCode(t *testing.T) {
	db := newTestDB(t)

	if err := db.CreateConversation("conv-1", "same-code", "First"); err != nil {
		t.Fatalf("first CreateConversation: %v", err)
	}
	err := db.CreateConversation("conv-2", "same-code", "Second")
	if err == nil {
		t.Fatal("expected error for duplicate invite code, got nil")
	}
}

func TestAddMemberAndGetConversations(t *testing.T) {
	db := newTestDB(t)

	if err := db.CreateConversation("conv-1", "invite-abc", "Test Group"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if err := db.AddMember("conv-1", "alice"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	conversations, err := db.GetConversationsByUsername("alice")
	if err != nil {
		t.Fatalf("GetConversationsByUsername: %v", err)
	}
	if len(conversations) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(conversations))
	}
	if conversations[0].ID != "conv-1" {
		t.Errorf("expected conv-1, got %q", conversations[0].ID)
	}
}

func TestAddMemberIdempotent(t *testing.T) {
	db := newTestDB(t)

	if err := db.CreateConversation("conv-1", "invite-abc", "Test Group"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if err := db.AddMember("conv-1", "alice"); err != nil {
		t.Fatalf("first AddMember: %v", err)
	}
	// Adding again should not error (INSERT OR IGNORE)
	if err := db.AddMember("conv-1", "alice"); err != nil {
		t.Fatalf("second AddMember: %v", err)
	}
}

func TestIsMember(t *testing.T) {
	db := newTestDB(t)

	if err := db.CreateConversation("conv-1", "invite-abc", "Test Group"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	is, err := db.IsMember("conv-1", "alice")
	if err != nil {
		t.Fatalf("IsMember: %v", err)
	}
	if is {
		t.Fatal("alice should not be a member yet")
	}

	if err := db.AddMember("conv-1", "alice"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	is, err = db.IsMember("conv-1", "alice")
	if err != nil {
		t.Fatalf("IsMember: %v", err)
	}
	if !is {
		t.Fatal("alice should be a member")
	}
}

func TestCreateAndListVideos(t *testing.T) {
	db := newTestDB(t)

	if err := db.CreateConversation("conv-1", "invite-abc", "Test Group"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if err := db.CreateVideo("vid-1", "conv-1", "alice", "/videos/conv-1/vid-1.mp4"); err != nil {
		t.Fatalf("CreateVideo: %v", err)
	}

	videos, err := db.GetVideosByConversation("conv-1")
	if err != nil {
		t.Fatalf("GetVideosByConversation: %v", err)
	}
	if len(videos) != 1 {
		t.Fatalf("expected 1 video, got %d", len(videos))
	}
	if videos[0].ID != "vid-1" {
		t.Errorf("expected vid-1, got %q", videos[0].ID)
	}
	if videos[0].Status != "pending" {
		t.Errorf("expected status 'pending', got %q", videos[0].Status)
	}
}

func TestUpdateVideoStatus(t *testing.T) {
	db := newTestDB(t)

	if err := db.CreateConversation("conv-1", "invite-abc", "Test Group"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if err := db.CreateVideo("vid-1", "conv-1", "alice", "/videos/conv-1/vid-1.mp4"); err != nil {
		t.Fatalf("CreateVideo: %v", err)
	}

	if err := db.UpdateVideoStatus("vid-1", "ready"); err != nil {
		t.Fatalf("UpdateVideoStatus: %v", err)
	}

	videos, err := db.GetVideosByConversation("conv-1")
	if err != nil {
		t.Fatalf("GetVideosByConversation: %v", err)
	}
	if videos[0].Status != "ready" {
		t.Errorf("expected status 'ready', got %q", videos[0].Status)
	}
}

func TestGetVideosEmpty(t *testing.T) {
	db := newTestDB(t)

	if err := db.CreateConversation("conv-1", "invite-abc", "Test Group"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	videos, err := db.GetVideosByConversation("conv-1")
	if err != nil {
		t.Fatalf("GetVideosByConversation: %v", err)
	}
	if len(videos) != 0 {
		t.Errorf("expected 0 videos, got %d", len(videos))
	}
}
