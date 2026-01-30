package api

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"pushem/internal/db"
	"pushem/internal/validation"
	"pushem/internal/webpush"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	db                 *db.DB
	webpush            *webpush.Service
	adminPasswordHash  []byte
	jwtSecret          []byte
	tokenExpiryMinutes int
	loginRateLimiter   *LoginRateLimiter
}

// LoginAttempt tracks a single login attempt
type LoginAttempt struct {
	Timestamp time.Time
}

// LoginRateLimiter tracks failed login attempts by IP address
type LoginRateLimiter struct {
	attempts       map[string][]LoginAttempt
	mu             sync.Mutex
	maxAttempts    int
	windowDuration time.Duration
}

// NewLoginRateLimiter creates a new rate limiter for login attempts
func NewLoginRateLimiter(maxAttempts int, windowMinutes int) *LoginRateLimiter {
	limiter := &LoginRateLimiter{
		attempts:       make(map[string][]LoginAttempt),
		maxAttempts:    maxAttempts,
		windowDuration: time.Duration(windowMinutes) * time.Minute,
	}

	// Start cleanup goroutine to remove old entries
	go limiter.cleanup()

	return limiter
}

// cleanup periodically removes expired entries from the rate limiter
func (l *LoginRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for ip, attempts := range l.attempts {
			// Filter out old attempts
			var validAttempts []LoginAttempt
			for _, attempt := range attempts {
				if now.Sub(attempt.Timestamp) < l.windowDuration {
					validAttempts = append(validAttempts, attempt)
				}
			}

			if len(validAttempts) == 0 {
				delete(l.attempts, ip)
			} else {
				l.attempts[ip] = validAttempts
			}
		}
		l.mu.Unlock()
	}
}

// IsAllowed checks if an IP address is allowed to attempt login
func (l *LoginRateLimiter) IsAllowed(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	attempts := l.attempts[ip]

	// Count valid attempts within the time window
	validAttempts := 0
	for _, attempt := range attempts {
		if now.Sub(attempt.Timestamp) < l.windowDuration {
			validAttempts++
		}
	}

	return validAttempts < l.maxAttempts
}

// RecordFailedAttempt records a failed login attempt for an IP address
func (l *LoginRateLimiter) RecordFailedAttempt(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.attempts[ip] = append(l.attempts[ip], LoginAttempt{
		Timestamp: time.Now(),
	})
}

// ResetAttempts clears all failed attempts for an IP address (called on successful login)
func (l *LoginRateLimiter) ResetAttempts(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, ip)
}

func NewHandler(database *db.DB, webpushService *webpush.Service, adminPassword string, tokenExpiryMinutes int, maxLoginAttempts int, loginRateLimitWindow int) *Handler {
	var adminPasswordHash []byte

	// Hash the admin password if provided
	if adminPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Warning: Failed to hash admin password: %v", err)
		} else {
			adminPasswordHash = hash
		}
	}

	// Generate a random JWT secret key
	jwtSecret := make([]byte, 32)
	if _, err := rand.Read(jwtSecret); err != nil {
		log.Printf("Warning: Failed to generate JWT secret: %v", err)
	}

	if tokenExpiryMinutes <= 0 {
		tokenExpiryMinutes = 60 // Default to 1 hour
	}

	if maxLoginAttempts <= 0 {
		maxLoginAttempts = 5 // Default to 5 attempts
	}

	if loginRateLimitWindow <= 0 {
		loginRateLimitWindow = 15 // Default to 15 minutes
	}

	// Create rate limiter for login attempts
	rateLimiter := NewLoginRateLimiter(maxLoginAttempts, loginRateLimitWindow)

	return &Handler{
		db:                 database,
		webpush:            webpushService,
		adminPasswordHash:  adminPasswordHash,
		jwtSecret:          jwtSecret,
		tokenExpiryMinutes: tokenExpiryMinutes,
		loginRateLimiter:   rateLimiter,
	}
}

// AdminClaims represents the JWT claims for admin authentication
type AdminClaims struct {
	Admin bool `json:"admin"`
	jwt.RegisteredClaims
}

// generateAdminToken creates a new JWT token for admin access
func (h *Handler) generateAdminToken() (string, error) {
	expirationTime := time.Now().Add(time.Duration(h.tokenExpiryMinutes) * time.Minute)

	claims := &AdminClaims{
		Admin: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "pushem",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(h.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// validateAdminToken validates a JWT token and returns whether it's valid
func (h *Handler) validateAdminToken(tokenString string) bool {
	claims := &AdminClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid || !claims.Admin {
		return false
	}

	return true
}

func (h *Handler) GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"publicKey": h.webpush.GetPublicKey(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type SubscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

type ProtectTopicRequest struct {
	Secret string `json:"secret"`
}

func (h *Handler) checkAuth(w http.ResponseWriter, r *http.Request, topic string) bool {
	// Check header first
	providedKey := r.Header.Get("X-Pushem-Key")
	if providedKey == "" {
		// Fallback to query param
		providedKey = r.URL.Query().Get("key")
	}

	// Verify the secret using bcrypt (includes timing attack protection)
	isValid, err := h.db.VerifyTopicSecret(topic, providedKey)
	if err != nil {
		log.Printf("Failed to verify topic secret: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return false
	}

	if !isValid {
		http.Error(w, "unauthorized: topic is protected", http.StatusUnauthorized)
		return false
	}

	return true
}

func (h *Handler) ProtectTopic(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		log.Printf("ProtectTopic: topic is empty")
		http.Error(w, "topic is required", http.StatusBadRequest)
		return
	}

	// Validate topic name
	if err := validation.ValidateTopic(topic); err != nil {
		log.Printf("ProtectTopic: invalid topic '%s': %v", topic, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req ProtectTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("ProtectTopic: failed to decode request body: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("ProtectTopic: topic='%s', secret length=%d (before sanitization)", topic, len(req.Secret))

	// Sanitize and validate secret
	req.Secret = validation.SanitizeString(req.Secret)
	log.Printf("ProtectTopic: secret length=%d (after sanitization)", len(req.Secret))

	if err := validation.ValidateSecret(req.Secret); err != nil {
		log.Printf("ProtectTopic: secret validation failed for topic '%s': %v", topic, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If already protected, check if authorized to change it
	if !h.checkAuth(w, r, topic) {
		log.Printf("ProtectTopic: auth check failed for topic '%s'", topic)
		return
	}

	if err := h.db.ProtectTopic(topic, req.Secret); err != nil {
		log.Printf("Failed to protect topic '%s': %v", topic, err)
		http.Error(w, "failed to protect topic", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully protected topic '%s'", topic)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "topic protected"})
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		http.Error(w, "topic is required", http.StatusBadRequest)
		return
	}

	// Validate topic name
	if err := validation.ValidateTopic(topic); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Authorization check
	if !h.checkAuth(w, r, topic) {
		return
	}

	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		http.Error(w, "endpoint, p256dh, and auth are required", http.StatusBadRequest)
		return
	}

	// Validate endpoint URL (SSRF protection)
	if err := validation.ValidateURL(req.Endpoint); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.db.SaveSubscription(topic, req.Endpoint, req.Keys.P256dh, req.Keys.Auth); err != nil {
		log.Printf("Failed to save subscription: %v", err)
		http.Error(w, "failed to save subscription", http.StatusInternalServerError)
		return
	}

	log.Printf("Subscribed to topic '%s': %s", topic, req.Endpoint)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "subscribed"})
}

func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		http.Error(w, "topic is required", http.StatusBadRequest)
		return
	}

	// Validate topic name
	if err := validation.ValidateTopic(topic); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Authorization check
	if !h.checkAuth(w, r, topic) {
		return
	}

	// Limit request body size to prevent DoS (10 MB max)
	const MaxBodySize = 10 * 1024 * 1024 // 10 MB
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySize)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Check if error is due to size limit
		if err.Error() == "http: request body too large" {
			http.Error(w, "request body too large (max 10 MB)", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
		}
		return
	}

	var payload webpush.NotificationPayload
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}
	} else {
		payload = webpush.NotificationPayload{
			Title:   "Notification",
			Message: string(body),
		}
	}

	if payload.Title == "" && payload.Message != "" {
		payload.Title = "Notification"
	}

	// Sanitize and validate message content
	payload.Title = validation.SanitizeString(payload.Title)
	payload.Message = validation.SanitizeString(payload.Message)
	if err := validation.ValidateMessage(payload.Title, payload.Message); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Save message to history
	if err := h.db.SaveMessage(topic, payload.Title, payload.Message); err != nil {
		log.Printf("Failed to save message to history: %v", err)
		// We initiate the publish anyway, even if saving history fails
	}

	subscriptions, err := h.db.GetSubscriptionsByTopic(topic)
	if err != nil {
		log.Printf("Failed to get subscriptions: %v", err)
		http.Error(w, "failed to get subscriptions", http.StatusInternalServerError)
		return
	}

	if len(subscriptions) == 0 {
		log.Printf("No subscriptions found for topic '%s'", topic)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "published",
			"sent":   0,
		})
		return
	}

	sent := 0
	failed := 0

	// Send notifications concurrently with limited parallelism
	const MaxConcurrentPushes = 10
	sem := make(chan struct{}, MaxConcurrentPushes)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, sub := range subscriptions {
		wg.Add(1)
		go func(s db.Subscription) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			err := h.webpush.SendNotification(s.Endpoint, s.P256dh, s.Auth, payload)
			if err != nil {
				log.Printf("Failed to send notification to %s: %v", s.Endpoint, err)

				if strings.Contains(err.Error(), "410 Gone") {
					log.Printf("Removing expired subscription: %s", s.Endpoint)
					if err := h.db.DeleteSubscription(s.Endpoint); err != nil {
						log.Printf("Failed to delete subscription: %v", err)
					}
				}

				mu.Lock()
				failed++
				mu.Unlock()
			} else {
				mu.Lock()
				sent++
				mu.Unlock()
			}
		}(sub)
	}

	wg.Wait()

	log.Printf("Published to topic '%s': sent=%d, failed=%d", topic, sent, failed)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "published",
		"sent":   sent,
		"failed": failed,
	})
}

func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		http.Error(w, "topic is required", http.StatusBadRequest)
		return
	}

	// Validate topic name
	if err := validation.ValidateTopic(topic); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Authorization check
	if !h.checkAuth(w, r, topic) {
		return
	}

	messages, err := h.db.GetMessagesByTopic(topic)
	if err != nil {
		log.Printf("Failed to get messages: %v", err)
		http.Error(w, "failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (h *Handler) ClearHistory(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		http.Error(w, "topic is required", http.StatusBadRequest)
		return
	}

	// Validate topic name
	if err := validation.ValidateTopic(topic); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Authorization check
	if !h.checkAuth(w, r, topic) {
		return
	}

	if err := h.db.ClearMessages(topic); err != nil {
		log.Printf("Failed to clear messages: %v", err)
		http.Error(w, "failed to clear messages", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "history cleared"})
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		http.Error(w, "topic is required", http.StatusBadRequest)
		return
	}

	messageIDStr := chi.URLParam(r, "messageId")
	if messageIDStr == "" {
		http.Error(w, "message ID is required", http.StatusBadRequest)
		return
	}

	// Validate topic name
	if err := validation.ValidateTopic(topic); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse message ID
	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil {
		http.Error(w, "invalid message ID", http.StatusBadRequest)
		return
	}

	// Authorization check
	if !h.checkAuth(w, r, topic) {
		return
	}

	// Delete the message
	if err := h.db.DeleteMessage(topic, messageID); err != nil {
		log.Printf("Failed to delete message %d from topic '%s': %v", messageID, topic, err)

		// Return appropriate status code based on error
		if err.Error() == "message not found" || err.Error() == "message does not belong to topic" {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, "failed to delete message", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Deleted message %d from topic '%s'", messageID, topic)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "message deleted"})
}

// Admin authentication middleware - validates JWT tokens
func (h *Handler) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If no admin password hash is set, admin panel is disabled
		if len(h.adminPasswordHash) == 0 {
			http.Error(w, "admin panel is disabled", http.StatusForbidden)
			return
		}

		// Get token from Authorization header (Bearer token)
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "unauthorized: invalid authorization header", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate the token
		if !h.validateAdminToken(token) {
			http.Error(w, "unauthorized: invalid or expired token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Admin: List all topics
func (h *Handler) AdminListTopics(w http.ResponseWriter, r *http.Request) {
	topics, err := h.db.ListAllTopics()
	if err != nil {
		log.Printf("Failed to list topics: %v", err)
		http.Error(w, "failed to list topics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(topics)
}

// Admin: Delete a topic and all its subscriptions
func (h *Handler) AdminDeleteTopic(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		http.Error(w, "topic is required", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteTopic(topic); err != nil {
		log.Printf("Failed to delete topic: %v", err)
		http.Error(w, "failed to delete topic", http.StatusInternalServerError)
		return
	}

	log.Printf("Admin: Deleted topic '%s'", topic)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "topic deleted"})
}

// Admin: Remove protection from a topic
func (h *Handler) AdminUnprotectTopic(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		http.Error(w, "topic is required", http.StatusBadRequest)
		return
	}

	if err := h.db.UnprotectTopic(topic); err != nil {
		log.Printf("Failed to unprotect topic: %v", err)
		http.Error(w, "failed to unprotect topic", http.StatusInternalServerError)
		return
	}

	log.Printf("Admin: Unprotected topic '%s'", topic)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "topic unprotected"})
}

// AdminLogin handles admin authentication and issues JWT tokens
func (h *Handler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	// If no admin password hash is set, admin panel is disabled
	if len(h.adminPasswordHash) == 0 {
		http.Error(w, "admin panel is disabled", http.StatusForbidden)
		return
	}

	// Get client IP address for rate limiting
	clientIP := getClientIP(r)

	// Check rate limiting
	if !h.loginRateLimiter.IsAllowed(clientIP) {
		log.Printf("Admin login rate limit exceeded for IP: %s", clientIP)
		http.Error(w, "too many failed login attempts, please try again later", http.StatusTooManyRequests)
		return
	}

	// Parse request body
	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Verify password using bcrypt
	err := bcrypt.CompareHashAndPassword(h.adminPasswordHash, []byte(req.Password))
	if err != nil {
		// Password doesn't match - record failed attempt
		h.loginRateLimiter.RecordFailedAttempt(clientIP)
		log.Printf("Admin login attempt with incorrect password from IP: %s", clientIP)
		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := h.generateAdminToken()
	if err != nil {
		log.Printf("Failed to generate admin token: %v", err)
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	// Successful login - reset rate limit for this IP
	h.loginRateLimiter.ResetAttempts(clientIP)
	log.Printf("Admin login successful from IP: %s", clientIP)

	// Return token and expiry info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":       token,
		"expires_in":  h.tokenExpiryMinutes * 60, // Return seconds
		"token_type":  "Bearer",
	})
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP if multiple are present
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
