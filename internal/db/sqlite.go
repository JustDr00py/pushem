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
	`
	_, err := db.conn.Exec(query)
	return err
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

func (db *DB) Close() error {
	return db.conn.Close()
}
