package storage

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func New(path string) (*DB, error) {
	slog.Info("opening database", "path", path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	slog.Info("database ready")
	return &DB{db}, nil
}

func migrate(db *sql.DB) error {
	slog.Info("running database migrations")

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS conversations (
			id          TEXT PRIMARY KEY,
			invite_code TEXT UNIQUE NOT NULL,
			name        TEXT NOT NULL,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS members (
			conversation_id TEXT NOT NULL,
			username        TEXT NOT NULL,
			joined_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (conversation_id, username),
			FOREIGN KEY (conversation_id) REFERENCES conversations(id)
		);

		CREATE TABLE IF NOT EXISTS videos (
			id              TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			uploader        TEXT NOT NULL,
			filename        TEXT NOT NULL,
			status          TEXT NOT NULL DEFAULT 'pending',
			uploaded_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id)
		);
	`)
	if err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	slog.Info("migrations complete")
	return nil
}
