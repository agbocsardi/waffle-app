package storage

import (
	"fmt"
	"time"
)

type Video struct {
	ID             string
	ConversationID string
	Uploader       string
	Filename       string
	Status         string // "pending", "ready", "error"
	UploadedAt     time.Time
}

func (db *DB) CreateVideo(id, conversationID, uploader, filename string) error {
	_, err := db.Exec(
		`INSERT INTO videos (id, conversation_id, uploader, filename, status) VALUES (?, ?, ?, ?, 'pending')`,
		id, conversationID, uploader, filename,
	)
	if err != nil {
		return fmt.Errorf("create video: %w", err)
	}
	return nil
}

func (db *DB) UpdateVideoStatus(id, status string) error {
	_, err := db.Exec(
		`UPDATE videos SET status = ? WHERE id = ?`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update video status: %w", err)
	}
	return nil
}

func (db *DB) GetVideosByConversation(conversationID string) ([]Video, error) {
	rows, err := db.Query(`
		SELECT id, conversation_id, uploader, filename, status, uploaded_at
		FROM videos
		WHERE conversation_id = ?
		ORDER BY uploaded_at DESC
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get videos by conversation: %w", err)
	}
	defer rows.Close()

	var videos []Video
	for rows.Next() {
		v := Video{}
		if err := rows.Scan(&v.ID, &v.ConversationID, &v.Uploader, &v.Filename, &v.Status, &v.UploadedAt); err != nil {
			return nil, fmt.Errorf("scan video: %w", err)
		}
		videos = append(videos, v)
	}
	return videos, nil
}
