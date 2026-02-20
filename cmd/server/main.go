package main

import (
	"log/slog"
	"net/http"
	"os"
	"waffle-app/internal/auth"
	"waffle-app/internal/conversations"
	"waffle-app/internal/storage"
	"waffle-app/internal/videos"
)

const (
	addr      = ":8080"
	dbPath    = "./waffle.db"
	videosDir = "./videos"
)

func main() {
	// Structured JSON logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("starting waffle server", "addr", addr)

	// Create videos directory
	if err := os.MkdirAll(videosDir, 0755); err != nil {
		slog.Error("failed to create videos directory", "error", err)
		os.Exit(1)
	}

	// Initialize database
	db, err := storage.New(dbPath)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize session store
	sessions := auth.NewStore()

	// Initialize handlers
	convHandler := conversations.NewHandler(db, sessions)
	videoHandler := videos.NewHandler(db, sessions, videosDir)

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/conversations/join", convHandler.Join)
	mux.HandleFunc("POST /api/conversations", convHandler.Create)
	mux.HandleFunc("GET /api/conversations", convHandler.List)
	mux.HandleFunc("POST /api/upload", videoHandler.Upload)
	mux.HandleFunc("GET /api/videos", videoHandler.List)

	// Serve static files
	mux.Handle("/", http.FileServer(http.Dir("web")))

	slog.Info("server listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
