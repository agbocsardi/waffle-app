package storage

import (
	"database/sql"
	"fmt"
	"time"
)

type Conversation struct {
	ID         string
	InviteCode string
	Name       string
	CreatedAt  time.Time
}

func (db *DB) CreateConversation(id, inviteCode, name string) error {
	_, err := db.Exec(
		`INSERT INTO conversations (id, invite_code, name) VALUES (?, ?, ?)`,
		id, inviteCode, name,
	)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	return nil
}

func (db *DB) GetConversationByInviteCode(inviteCode string) (*Conversation, error) {
	row := db.QueryRow(
		`SELECT id, invite_code, name, created_at FROM conversations WHERE invite_code = ?`,
		inviteCode,
	)
	c := &Conversation{}
	err := row.Scan(&c.ID, &c.InviteCode, &c.Name, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get conversation by invite code: %w", err)
	}
	return c, nil
}

func (db *DB) GetConversationsByUsername(username string) ([]Conversation, error) {
	rows, err := db.Query(`
		SELECT c.id, c.invite_code, c.name, c.created_at
		FROM conversations c
		JOIN members m ON c.id = m.conversation_id
		WHERE m.username = ?
		ORDER BY c.created_at DESC
	`, username)
	if err != nil {
		return nil, fmt.Errorf("get conversations by username: %w", err)
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		c := Conversation{}
		if err := rows.Scan(&c.ID, &c.InviteCode, &c.Name, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		conversations = append(conversations, c)
	}
	return conversations, nil
}

func (db *DB) IsMember(conversationID, username string) (bool, error) {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM members WHERE conversation_id = ? AND username = ?`,
		conversationID, username,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return count > 0, nil
}

func (db *DB) AddMember(conversationID, username string) error {
	_, err := db.Exec(
		`INSERT OR IGNORE INTO members (conversation_id, username) VALUES (?, ?)`,
		conversationID, username,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}
