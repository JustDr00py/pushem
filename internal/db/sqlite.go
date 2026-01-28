package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
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
	// Hash the secret using bcrypt before storing
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash secret: %w", err)
	}

	query := `
	INSERT INTO topics (topic, secret)
	VALUES (?, ?)
	ON CONFLICT(topic) DO UPDATE SET
		secret = excluded.secret
	`
	_, err = db.conn.Exec(query, topic, string(hashedSecret))
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

// VerifyTopicSecret checks if the provided secret matches the stored hashed secret for a topic.
// Returns true if the topic is public (no secret set) or if the secret matches.
// Returns false if the secret doesn't match or if there's an error.
func (db *DB) VerifyTopicSecret(topic, providedSecret string) (bool, error) {
	hashedSecret, err := db.GetTopicSecret(topic)
	if err != nil {
		return false, fmt.Errorf("failed to get topic secret: %w", err)
	}

	// Topic is public if no secret is set
	if hashedSecret == "" {
		return true, nil
	}

	// Compare the provided secret with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(hashedSecret), []byte(providedSecret))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, fmt.Errorf("failed to verify secret: %w", err)
	}

	return true, nil
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

func (db *DB) DeleteOldMessages(daysOld int) (int64, error) {
	query := `DELETE FROM messages WHERE created_at < datetime('now', ?)`
	result, err := db.conn.Exec(query, fmt.Sprintf("-%d days", daysOld))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) GetMessageCount() (int64, error) {
	var count int64
	err := db.conn.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
	return count, err
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// TopicInfo contains information about a topic for admin purposes
type TopicInfo struct {
	Name              string `json:"name"`
	IsProtected       bool   `json:"is_protected"`
	SubscriptionCount int    `json:"subscription_count"`
	MessageCount      int    `json:"message_count"`
	CreatedAt         string `json:"created_at,omitempty"`
}

// ListAllTopics returns all topics with their metadata
func (db *DB) ListAllTopics() ([]TopicInfo, error) {
	// Get all unique topics from subscriptions
	query := `
		SELECT DISTINCT s.topic,
			CASE WHEN t.secret IS NOT NULL THEN 1 ELSE 0 END as is_protected,
			COUNT(s.id) as subscription_count,
			t.created_at
		FROM subscriptions s
		LEFT JOIN topics t ON s.topic = t.topic
		GROUP BY s.topic
		ORDER BY s.topic ASC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []TopicInfo
	for rows.Next() {
		var info TopicInfo
		var createdAt sql.NullString
		if err := rows.Scan(&info.Name, &info.IsProtected, &info.SubscriptionCount, &createdAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			info.CreatedAt = createdAt.String
		}

		// Get message count for this topic
		var msgCount int
		msgErr := db.conn.QueryRow("SELECT COUNT(*) FROM messages WHERE topic = ?", info.Name).Scan(&msgCount)
		if msgErr == nil {
			info.MessageCount = msgCount
		}

		topics = append(topics, info)
	}

	return topics, rows.Err()
}

// DeleteTopic removes a topic and all associated data
func (db *DB) DeleteTopic(topic string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete subscriptions
	if _, err := tx.Exec("DELETE FROM subscriptions WHERE topic = ?", topic); err != nil {
		return fmt.Errorf("failed to delete subscriptions: %w", err)
	}

	// Delete messages
	if _, err := tx.Exec("DELETE FROM messages WHERE topic = ?", topic); err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	// Delete topic protection
	if _, err := tx.Exec("DELETE FROM topics WHERE topic = ?", topic); err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}

	return tx.Commit()
}

// UnprotectTopic removes protection from a topic
func (db *DB) UnprotectTopic(topic string) error {
	query := `DELETE FROM topics WHERE topic = ?`
	_, err := db.conn.Exec(query, topic)
	return err
}
