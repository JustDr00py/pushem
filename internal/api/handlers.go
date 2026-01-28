package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"pushem/internal/db"
	"pushem/internal/webpush"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db      *db.DB
	webpush *webpush.Service
}

func NewHandler(database *db.DB, webpushService *webpush.Service) *Handler {
	return &Handler{
		db:      database,
		webpush: webpushService,
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

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		http.Error(w, "topic is required", http.StatusBadRequest)
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
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

	for _, sub := range subscriptions {
		err := h.webpush.SendNotification(sub.Endpoint, sub.P256dh, sub.Auth, payload)
		if err != nil {
			log.Printf("Failed to send notification to %s: %v", sub.Endpoint, err)

			if strings.Contains(err.Error(), "410 Gone") {
				log.Printf("Removing expired subscription: %s", sub.Endpoint)
				if err := h.db.DeleteSubscription(sub.Endpoint); err != nil {
					log.Printf("Failed to delete subscription: %v", err)
				}
			}
			failed++
		} else {
			sent++
		}
	}

	log.Printf("Published to topic '%s': sent=%d, failed=%d", topic, sent, failed)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "published",
		"sent":   sent,
		"failed": failed,
	})
}
