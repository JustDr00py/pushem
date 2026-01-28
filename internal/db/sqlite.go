package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

type Subscription struct {
	ID       int
	Topic    string
	Endpoint string
	P256dh   string
	Auth     string
}

func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS subscriptions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		topic TEXT NOT NULL,
		endpoint TEXT NOT NULL,
		p256dh TEXT NOT NULL,
		auth TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(topic, endpoint)
	);
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		topic TEXT NOT NULL,
		title TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS topics (
		topic TEXT PRIMARY KEY,
		secret TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) ProtectTopic(topic, secret string) error {
	query := `
	INSERT INTO topics (topic, secret)
	VALUES (?, ?)
	ON CONFLICT(topic) DO UPDATE SET
		secret = excluded.secret
	`
	_, err := db.conn.Exec(query, topic, secret)
	return err
}

func (db *DB) GetTopicSecret(topic string) (string, error) {
	query := `SELECT secret FROM topics WHERE topic = ?`
	var secret string
	err := db.conn.QueryRow(query, topic).Scan(&secret)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return secret, err
}

func (db *DB) SaveSubscription(topic, endpoint, p256dh, auth string) error {
	query := `
	INSERT INTO subscriptions (topic, endpoint, p256dh, auth)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(topic, endpoint) DO UPDATE SET
		p256dh = excluded.p256dh,
		auth = excluded.auth
	`
	_, err := db.conn.Exec(query, topic, endpoint, p256dh, auth)
	return err
}

func (db *DB) GetSubscriptionsByTopic(topic string) ([]Subscription, error) {
	query := `SELECT id, topic, endpoint, p256dh, auth FROM subscriptions WHERE topic = ?`
	rows, err := db.conn.Query(query, topic)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.Topic, &sub.Endpoint, &sub.P256dh, &sub.Auth); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, rows.Err()
}

func (db *DB) DeleteSubscription(endpoint string) error {
	query := `DELETE FROM subscriptions WHERE endpoint = ?`
	_, err := db.conn.Exec(query, endpoint)
	return err
}

type Message struct {
	ID        int
	Topic     string
	Title     string
	Message   string
	CreatedAt string
}

func (db *DB) SaveMessage(topic, title, message string) error {
	query := `
	INSERT INTO messages (topic, title, message)
	VALUES (?, ?, ?)
	`
	_, err := db.conn.Exec(query, topic, title, message)
	return err
}

func (db *DB) GetMessagesByTopic(topic string) ([]Message, error) {
	query := `
	SELECT id, topic, title, message, created_at 
	FROM messages 
	WHERE topic = ? 
	ORDER BY created_at DESC 
	LIMIT 50`
	
	rows, err := db.conn.Query(query, topic)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.Topic, &msg.Title, &msg.Message, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func (db *DB) ClearMessages(topic string) error {
	query := `DELETE FROM messages WHERE topic = ?`
	_, err := db.conn.Exec(query, topic)
	return err
}

func (db *DB) Close() error {
	return db.conn.Close()
}
