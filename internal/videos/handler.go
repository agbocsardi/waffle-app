package videos

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"waffle-app/internal/auth"
	"waffle-app/internal/storage"
)

const (
	maxUploadSize = 500 << 20 // 500 MB
	maxRetries    = 3
	retryDelay    = 2 * time.Second
)

var allowedExtensions = map[string]bool{
	".mp4": true,
	".mov": true,
	".avi": true,
	".mkv": true,
}

type Handler struct {
	DB        *storage.DB
	Sessions  *auth.Store
	VideosDir string
}

func NewHandler(db *storage.DB, sessions *auth.Store, videosDir string) *Handler {
	return &Handler{DB: db, Sessions: sessions, VideosDir: videosDir}
}

// POST /api/upload
// Multipart form: file, conversation_id
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireSession(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		slog.Warn("failed to parse multipart form", "error", err)
		http.Error(w, "request too large or malformed", http.StatusBadRequest)
		return
	}

	conversationID := r.FormValue("conversation_id")
	if conversationID == "" {
		http.Error(w, "'conversation_id' is required", http.StatusBadRequest)
		return
	}

	// Verify user is a member of the conversation
	isMember, err := h.DB.IsMember(conversationID, session.Username)
	if err != nil {
		slog.Error("failed to check membership", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !isMember {
		slog.Warn("upload attempted by non-member", "username", session.Username, "conversation_id", conversationID)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		slog.Warn("unsupported file extension", "ext", ext, "username", session.Username)
		http.Error(w, fmt.Sprintf("unsupported file type: %s", ext), http.StatusBadRequest)
		return
	}

	videoID, err := generateID()
	if err != nil {
		slog.Error("failed to generate video id", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	convDir := filepath.Join(h.VideosDir, conversationID)
	if err := os.MkdirAll(convDir, 0755); err != nil {
		slog.Error("failed to create conversation directory", "error", err, "dir", convDir)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	originalPath := filepath.Join(convDir, "original_"+videoID+ext)
	outputPath := filepath.Join(convDir, videoID+".mp4")

	// Save original file to disk
	slog.Info("saving original upload", "path", originalPath, "username", session.Username)
	if err := saveFile(file, originalPath); err != nil {
		slog.Error("failed to save uploaded file", "error", err, "path", originalPath)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Record in DB as pending before transcoding
	if err := h.DB.CreateVideo(videoID, conversationID, session.Username, outputPath); err != nil {
		slog.Error("failed to create video record", "error", err)
		os.Remove(originalPath)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Transcode asynchronously so the client gets a fast response
	go h.transcode(videoID, originalPath, outputPath)

	slog.Info("upload accepted, transcoding started", "video_id", videoID, "username", session.Username)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"video_id": videoID,
		"status":   "pending",
	})
}

// GET /api/videos?conversation_id=...
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	session, ok := h.requireSession(w, r)
	if !ok {
		return
	}

	conversationID := r.URL.Query().Get("conversation_id")
	if conversationID == "" {
		http.Error(w, "'conversation_id' is required", http.StatusBadRequest)
		return
	}

	isMember, err := h.DB.IsMember(conversationID, session.Username)
	if err != nil {
		slog.Error("failed to check membership", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !isMember {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	videos, err := h.DB.GetVideosByConversation(conversationID)
	if err != nil {
		slog.Error("failed to list videos", "error", err, "conversation_id", conversationID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Debug("listed videos", "conversation_id", conversationID, "count", len(videos))

	type response struct {
		ID         string `json:"id"`
		Uploader   string `json:"uploader"`
		Status     string `json:"status"`
		UploadedAt string `json:"uploaded_at"`
	}
	result := make([]response, 0, len(videos))
	for _, v := range videos {
		result = append(result, response{
			ID:         v.ID,
			Uploader:   v.Uploader,
			Status:     v.Status,
			UploadedAt: v.UploadedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) transcode(videoID, inputPath, outputPath string) {
	slog.Info("starting transcoding", "video_id", videoID, "input", inputPath, "output", outputPath)

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		slog.Info("transcoding attempt", "video_id", videoID, "attempt", attempt, "max", maxRetries)

		cmd := exec.Command("ffmpeg",
			"-i", inputPath,
			"-vf", "scale=-2:720",
			"-c:v", "libx264",
			"-c:a", "aac",
			"-y", // overwrite output if exists
			outputPath,
		)

		output, err := cmd.CombinedOutput()
		if err == nil {
			slog.Info("transcoding succeeded", "video_id", videoID, "attempt", attempt)

			// Delete original only on success
			slog.Info("deleting original file", "path", inputPath)
			if err := os.Remove(inputPath); err != nil {
				slog.Error("failed to delete original file", "error", err, "path", inputPath)
			} else {
				slog.Info("original file deleted", "path", inputPath)
			}

			if err := h.DB.UpdateVideoStatus(videoID, "ready"); err != nil {
				slog.Error("failed to update video status to ready", "error", err, "video_id", videoID)
			}
			return
		}

		lastErr = err
		slog.Warn("transcoding attempt failed",
			"video_id", videoID,
			"attempt", attempt,
			"error", err,
			"ffmpeg_output", string(output),
		)

		if attempt < maxRetries {
			slog.Info("waiting before retry", "delay", retryDelay, "video_id", videoID)
			time.Sleep(retryDelay)
		}
	}

	slog.Error("transcoding failed after all retries, original file retained",
		"video_id", videoID,
		"input", inputPath,
		"error", lastErr,
	)
	if err := h.DB.UpdateVideoStatus(videoID, "error"); err != nil {
		slog.Error("failed to update video status to error", "error", err, "video_id", videoID)
	}
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

func saveFile(src io.Reader, destPath string) error {
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, src); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
