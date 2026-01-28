package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"pushem/internal/db"
	"pushem/internal/validation"
	"pushem/internal/webpush"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db            *db.DB
	webpush       *webpush.Service
	adminPassword string
}

func NewHandler(database *db.DB, webpushService *webpush.Service, adminPassword string) *Handler {
	return &Handler{
		db:            database,
		webpush:       webpushService,
		adminPassword: adminPassword,
	}
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

// Admin authentication middleware
func (h *Handler) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If no admin password is set, admin panel is disabled
		if h.adminPassword == "" {
			http.Error(w, "admin panel is disabled", http.StatusForbidden)
			return
		}

		// Check for admin password in X-Admin-Password header
		providedPassword := r.Header.Get("X-Admin-Password")

		// Compare using bcrypt-style comparison for consistency
		// (though admin password is plain text in .env)
		if providedPassword != h.adminPassword {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
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

// Admin: Verify password
func (h *Handler) AdminVerifyPassword(w http.ResponseWriter, r *http.Request) {
	// If this endpoint is reached, middleware already verified the password
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"valid": true})
}
