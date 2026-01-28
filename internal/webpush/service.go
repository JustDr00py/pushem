package webpush

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"
)

const (
	vapidKeysFile = "vapid_keys.json"
)

type Service struct {
	privateKey string
	publicKey  string
}

type VAPIDKeys struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

func NewService() (*Service, error) {
	keys, err := loadOrGenerateKeys()
	if err != nil {
		return nil, err
	}

	return &Service{
		privateKey: keys.PrivateKey,
		publicKey:  keys.PublicKey,
	}, nil
}

func loadOrGenerateKeys() (*VAPIDKeys, error) {
	if _, err := os.Stat(vapidKeysFile); err == nil {
		data, err := os.ReadFile(vapidKeysFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read VAPID keys file: %w", err)
		}

		var keys VAPIDKeys
		if err := json.Unmarshal(data, &keys); err != nil {
			return nil, fmt.Errorf("failed to unmarshal VAPID keys: %w", err)
		}

		log.Printf("Loaded existing VAPID keys")
		return &keys, nil
	}

	log.Println("Generating new VAPID keys...")
	privateKey, publicKey, err := generateVAPIDKeys()
	if err != nil {
		return nil, err
	}

	keys := &VAPIDKeys{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal VAPID keys: %w", err)
	}

	if err := os.WriteFile(vapidKeysFile, data, 0600); err != nil {
		return nil, fmt.Errorf("failed to write VAPID keys file: %w", err)
	}

	log.Printf("Generated new VAPID keys and saved to %s", vapidKeysFile)
	log.Printf("Public Key: %s", publicKey)

	return keys, nil
}

func generateVAPIDKeys() (privateKey, publicKey string, err error) {
	// Generate ECDSA P-256 key pair
	curve := elliptic.P256()
	privKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate ECDSA key: %w", err)
	}

	// Extract raw private key bytes (D value) - must be exactly 32 bytes for P-256
	privBytes := privKey.D.Bytes()
	if len(privBytes) < 32 {
		// Pad with leading zeros if needed
		padded := make([]byte, 32)
		copy(padded[32-len(privBytes):], privBytes)
		privBytes = padded
	}

	// Extract public key in uncompressed format (0x04 + X + Y)
	pubBytes := elliptic.Marshal(curve, privKey.PublicKey.X, privKey.PublicKey.Y)

	// Encode as base64url
	privateKey = base64.RawURLEncoding.EncodeToString(privBytes)
	publicKey = base64.RawURLEncoding.EncodeToString(pubBytes)

	return privateKey, publicKey, nil
}

func (s *Service) GetPublicKey() string {
	return s.publicKey
}

type NotificationPayload struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	ClickURL string `json:"click_url,omitempty"`
}

func (s *Service) SendNotification(endpoint, p256dh, auth string, payload NotificationPayload) error {
	sub := &webpush.Subscription{
		Endpoint: endpoint,
		Keys: webpush.Keys{
			P256dh: p256dh,
			Auth:   auth,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := webpush.SendNotification(payloadBytes, sub, &webpush.Options{
		Subscriber:      "mailto:admin@example.com",
		VAPIDPrivateKey: s.privateKey,
		VAPIDPublicKey:  s.publicKey,
		TTL:             86400,
	})
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 410 {
		return fmt.Errorf("subscription expired (410 Gone)")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("push service returned status: %d", resp.StatusCode)
	}

	return nil
}
