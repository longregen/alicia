package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type ArchivedMessage struct {
	ID         string
	ChatJID    string
	SenderJID  string
	SenderName string
	Content    string
	MediaType  string
	Timestamp  time.Time
	IsFromMe   bool
	IsGroup    bool
}

type Archive struct {
	db *sql.DB
}

func NewArchive(dbPath string) (*Archive, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open archive db: %w", err)
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init archive schema: %w", err)
	}

	return &Archive{db: db}, nil
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			chat_jid TEXT NOT NULL,
			sender_jid TEXT NOT NULL,
			sender_name TEXT DEFAULT '',
			content TEXT DEFAULT '',
			media_type TEXT DEFAULT '',
			timestamp INTEGER NOT NULL,
			is_from_me INTEGER DEFAULT 0,
			is_group INTEGER DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_messages_chat ON messages(chat_jid);
		CREATE INDEX IF NOT EXISTS idx_messages_ts ON messages(timestamp);

		CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
			content, sender_name, chat_jid,
			content='messages', content_rowid='rowid'
		);

		-- Triggers to keep FTS in sync
		CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
			INSERT INTO messages_fts(rowid, content, sender_name, chat_jid)
			VALUES (new.rowid, new.content, new.sender_name, new.chat_jid);
		END;

		CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
			INSERT INTO messages_fts(messages_fts, rowid, content, sender_name, chat_jid)
			VALUES ('delete', old.rowid, old.content, old.sender_name, old.chat_jid);
		END;

		CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
			INSERT INTO messages_fts(messages_fts, rowid, content, sender_name, chat_jid)
			VALUES ('delete', old.rowid, old.content, old.sender_name, old.chat_jid);
			INSERT INTO messages_fts(rowid, content, sender_name, chat_jid)
			VALUES (new.rowid, new.content, new.sender_name, new.chat_jid);
		END;

		CREATE TABLE IF NOT EXISTS state (
			key TEXT PRIMARY KEY,
			value TEXT
		);
	`)
	return err
}

func (a *Archive) Store(msg *ArchivedMessage) error {
	_, err := a.db.Exec(`
		INSERT OR IGNORE INTO messages (id, chat_jid, sender_jid, sender_name, content, media_type, timestamp, is_from_me, is_group)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.ChatJID, msg.SenderJID, msg.SenderName, msg.Content, msg.MediaType,
		msg.Timestamp.Unix(), boolToInt(msg.IsFromMe), boolToInt(msg.IsGroup),
	)
	if err != nil {
		return fmt.Errorf("archive store: %w", err)
	}
	return nil
}

func (a *Archive) GetState(key string) (string, error) {
	var value string
	err := a.db.QueryRow("SELECT value FROM state WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (a *Archive) SetState(key, value string) error {
	_, err := a.db.Exec("INSERT OR REPLACE INTO state (key, value) VALUES (?, ?)", key, value)
	return err
}

func (a *Archive) Close() error {
	return a.db.Close()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
