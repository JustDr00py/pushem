package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	MaxTopicLength   = 100
	MaxMessageLength = 4096
	MaxTitleLength   = 256
	MaxSecretLength  = 256
	MinSecretLength  = 8
)

var (
	// Topic names: alphanumeric, hyphens, underscores, dots
	topicRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

	// Forbidden topic names (reserved for system use)
	forbiddenTopics = map[string]bool{
		"admin":   true,
		"system":  true,
		"api":     true,
		"vapid":   true,
		"health":  true,
		"metrics": true,
	}
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateTopic validates topic name
func ValidateTopic(topic string) error {
	if topic == "" {
		return &ValidationError{"topic", "topic cannot be empty"}
	}

	if len(topic) > MaxTopicLength {
		return &ValidationError{"topic", fmt.Sprintf("topic must be at most %d characters", MaxTopicLength)}
	}

	if !utf8.ValidString(topic) {
		return &ValidationError{"topic", "topic contains invalid UTF-8"}
	}

	if !topicRegex.MatchString(topic) {
		return &ValidationError{"topic", "topic can only contain letters, numbers, hyphens, underscores, and dots"}
	}

	// Check for path traversal attempts
	if strings.Contains(topic, "..") || strings.Contains(topic, "//") {
		return &ValidationError{"topic", "topic contains invalid characters"}
	}

	// Check forbidden names
	if forbiddenTopics[strings.ToLower(topic)] {
		return &ValidationError{"topic", "topic name is reserved"}
	}

	return nil
}

// ValidateMessage validates notification message content
func ValidateMessage(title, message string) error {
	if message == "" {
		return &ValidationError{"message", "message cannot be empty"}
	}

	if len(title) > MaxTitleLength {
		return &ValidationError{"title", fmt.Sprintf("title must be at most %d characters", MaxTitleLength)}
	}

	if len(message) > MaxMessageLength {
		return &ValidationError{"message", fmt.Sprintf("message must be at most %d characters", MaxMessageLength)}
	}

	if !utf8.ValidString(title) {
		return &ValidationError{"title", "title contains invalid UTF-8"}
	}

	if !utf8.ValidString(message) {
		return &ValidationError{"message", "message contains invalid UTF-8"}
	}

	// Check for null bytes (can cause issues)
	if strings.Contains(title, "\x00") || strings.Contains(message, "\x00") {
		return &ValidationError{"message", "message contains null bytes"}
	}

	return nil
}

// ValidateSecret validates topic secret key
func ValidateSecret(secret string) error {
	if secret == "" {
		return &ValidationError{"secret", "secret cannot be empty"}
	}

	if len(secret) < MinSecretLength {
		return &ValidationError{"secret", fmt.Sprintf("secret must be at least %d characters", MinSecretLength)}
	}

	if len(secret) > MaxSecretLength {
		return &ValidationError{"secret", fmt.Sprintf("secret must be at most %d characters", MaxSecretLength)}
	}

	if !utf8.ValidString(secret) {
		return &ValidationError{"secret", "secret contains invalid UTF-8"}
	}

	// Warn about weak secrets (common patterns)
	lower := strings.ToLower(secret)
	if lower == "password" || lower == "12345678" || lower == "qwertyui" {
		return &ValidationError{"secret", "secret is too weak"}
	}

	return nil
}

// ValidateURL validates subscription endpoint URL
func ValidateURL(endpoint string) error {
	if endpoint == "" {
		return &ValidationError{"endpoint", "endpoint cannot be empty"}
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return &ValidationError{"endpoint", "invalid URL format"}
	}

	// Must be HTTPS for security
	if parsed.Scheme != "https" {
		return &ValidationError{"endpoint", "endpoint must use HTTPS"}
	}

	// Check for potentially dangerous hosts (SSRF protection)
	host := strings.ToLower(parsed.Hostname())

	// Block localhost and private IPs
	if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" ||
		strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "172.16.") ||
		strings.HasPrefix(host, "172.17.") ||
		strings.HasPrefix(host, "172.18.") ||
		strings.HasPrefix(host, "172.19.") ||
		strings.HasPrefix(host, "172.20.") ||
		strings.HasPrefix(host, "172.21.") ||
		strings.HasPrefix(host, "172.22.") ||
		strings.HasPrefix(host, "172.23.") ||
		strings.HasPrefix(host, "172.24.") ||
		strings.HasPrefix(host, "172.25.") ||
		strings.HasPrefix(host, "172.26.") ||
		strings.HasPrefix(host, "172.27.") ||
		strings.HasPrefix(host, "172.28.") ||
		strings.HasPrefix(host, "172.29.") ||
		strings.HasPrefix(host, "172.30.") ||
		strings.HasPrefix(host, "172.31.") {
		return &ValidationError{"endpoint", "endpoint must be a public URL"}
	}

	return nil
}

// SanitizeString removes potentially dangerous characters
func SanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")

	// Trim whitespace
	s = strings.TrimSpace(s)

	return s
}
