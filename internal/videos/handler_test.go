package videos_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"waffle-app/internal/auth"
	"waffle-app/internal/storage"
	"waffle-app/internal/videos"
)

func setupTest(t *testing.T) (*storage.DB, *auth.Store, string) {
	t.Helper()

	f, err := os.CreateTemp("", "waffle_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	db, err := storage.New(f.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	dir := t.TempDir()
	sessions := auth.NewStore()

	return db, sessions, dir
}

func authenticatedRequest(t *testing.T, sessions *auth.Store, method, target string, body *bytes.Buffer, contentType string) *http.Request {
	t.Helper()
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, target, body)
	} else {
		req, err = http.NewRequest(method, target, nil)
	}
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	token, err := sessions.Create("alice")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: "waffle_session", Value: token})
	return req
}

func TestUpload_UnsupportedFileType(t *testing.T) {
	db, sessions, dir := setupTest(t)

	if err := db.CreateConversation("conv-1", "invite-abc", "Test"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if err := db.AddMember("conv-1", "alice"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("conversation_id", "conv-1")
	part, _ := writer.CreateFormFile("file", "test.exe")
	part.Write([]byte("fake content"))
	writer.Close()

	req := authenticatedRequest(t, sessions, "POST", "/api/upload", body, writer.FormDataContentType())
	rr := httptest.NewRecorder()

	h := videos.NewHandler(db, sessions, dir)
	h.Upload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestUpload_NonMemberForbidden(t *testing.T) {
	db, sessions, dir := setupTest(t)

	if err := db.CreateConversation("conv-1", "invite-abc", "Test"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	// Alice is NOT added as a member

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("conversation_id", "conv-1")
	part, _ := writer.CreateFormFile("file", "test.mp4")
	part.Write([]byte("fake content"))
	writer.Close()

	req := authenticatedRequest(t, sessions, "POST", "/api/upload", body, writer.FormDataContentType())
	rr := httptest.NewRecorder()

	h := videos.NewHandler(db, sessions, dir)
	h.Upload(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestUpload_MissingConversationID(t *testing.T) {
	db, sessions, dir := setupTest(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.mp4")
	part.Write([]byte("fake content"))
	writer.Close()

	req := authenticatedRequest(t, sessions, "POST", "/api/upload", body, writer.FormDataContentType())
	rr := httptest.NewRecorder()

	h := videos.NewHandler(db, sessions, dir)
	h.Upload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestUpload_Unauthenticated(t *testing.T) {
	db, sessions, dir := setupTest(t)

	req, _ := http.NewRequest("POST", "/api/upload", nil)
	rr := httptest.NewRecorder()

	h := videos.NewHandler(db, sessions, dir)
	h.Upload(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestUpload_AcceptedAndOriginalSaved(t *testing.T) {
	db, sessions, dir := setupTest(t)

	if err := db.CreateConversation("conv-1", "invite-abc", "Test"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if err := db.AddMember("conv-1", "alice"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("conversation_id", "conv-1")
	part, _ := writer.CreateFormFile("file", "test.mp4")
	part.Write([]byte("fake video content"))
	writer.Close()

	req := authenticatedRequest(t, sessions, "POST", "/api/upload", body, writer.FormDataContentType())
	rr := httptest.NewRecorder()

	h := videos.NewHandler(db, sessions, dir)
	h.Upload(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["video_id"] == "" {
		t.Error("expected video_id in response")
	}
	if resp["status"] != "pending" {
		t.Errorf("expected status 'pending', got %q", resp["status"])
	}

	// Check original file was saved to disk
	convDir := filepath.Join(dir, "conv-1")
	entries, err := os.ReadDir(convDir)
	if err != nil {
		t.Fatalf("read conv dir: %v", err)
	}
	found := false
	for _, e := range entries {
		if len(e.Name()) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected original file to be saved in conversation directory")
	}
}

func TestList_Unauthenticated(t *testing.T) {
	db, sessions, dir := setupTest(t)

	req, _ := http.NewRequest("GET", "/api/videos?conversation_id=conv-1", nil)
	rr := httptest.NewRecorder()

	h := videos.NewHandler(db, sessions, dir)
	h.List(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestList_NonMemberForbidden(t *testing.T) {
	db, sessions, dir := setupTest(t)

	if err := db.CreateConversation("conv-1", "invite-abc", "Test"); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	// alice is NOT a member

	req := authenticatedRequest(t, sessions, "GET", "/api/videos?conversation_id=conv-1", nil, "")
	rr := httptest.NewRecorder()

	h := videos.NewHandler(db, sessions, dir)
	h.List(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
