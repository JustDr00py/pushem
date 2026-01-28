package webpush

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/golang-jwt/jwt/v5"
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

	// Check if endpoint is APNs (Apple Push Notification service)
	if strings.Contains(endpoint, "push.apple.com") {
		// Use custom handler for Apple to ensure proper JWT expiration (< 1h)
		return s.sendToApple(endpoint, p256dh, auth, payloadBytes)
	}
	
	subscriber := os.Getenv("VAPID_SUBJECT")
	if subscriber == "" {
		subscriber = "mailto:admin@pushem.local"
	}
	if !strings.HasPrefix(subscriber, "mailto:") {
		subscriber = "mailto:" + subscriber
	}

	resp, err := webpush.SendNotification(payloadBytes, sub, &webpush.Options{
		Subscriber:      subscriber,
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
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Push service error response: %s", string(body))
		return fmt.Errorf("push service returned status: %d", resp.StatusCode)
	}

	return nil
}


import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/golang-jwt/jwt/v5"
)

// ... existing code ...

type AppleTransport struct {
	Transport   http.RoundTripper
	PrivateKey  string
	PublicKey   string
	Subscriber  string
}

func (t *AppleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Generate new VAPID token with 1h expiration
	token, err := generateVAPIDToken(t.PrivateKey, t.Subscriber)
	if err != nil {
		return nil, fmt.Errorf("failed to generate apple vapid token: %w", err)
	}

	// Sign the token (we need to do this manually or use a helper)
	// Actually, generating the full Authorization header is easier.
	// Header format: vapid t=jwt, k=pubkey
	
	authHeader := fmt.Sprintf("vapid t=%s, k=%s", token, t.PublicKey)
	req.Header.Set("Authorization", authHeader)
	
	// Delegate to original transport
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return transport.RoundTrip(req)
}

func generateVAPIDToken(privateKeyStr, subscriber string) (string, error) {
	// Decode private key
	privBytes, err := base64.RawURLEncoding.DecodeString(privateKeyStr)
	if err != nil {
		return "", err
	}
	
	curve := elliptic.P256()
	x, y := curve.ScalarBaseMult(privBytes)
	privKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(privBytes),
	}

	// Create JWT
	now := time.Now()
	claims := jwt.MapClaims{
		"aud": "https://web.push.apple.com", // Apple requires the generic URL or specific? 
		// Actually "aud" should be the origin of the endpoint.
		// But in RoundTrip we know the request URL. 
		// Wait, "aud" must match the push service origin.
		// For Apple it is https://web.push.apple.com
		"exp": now.Add(time.Hour).Unix(), // 1 hour expiration
		"sub": subscriber,
	}
	
	// We need to set "aud" dynamically based on request, but here we are in a helper.
	// Let's move this logic to RoundTrip where we have req.URL.
	return "", nil 
}

// ... refactoring to put logic in RoundTrip ...

func (t *AppleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Origin of the push service
	origin := fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Host)
	
	tokenString, err := generateToken(t.PrivateKey, t.Subscriber, origin)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("vapid t=%s, k=%s", tokenString, t.PublicKey))

	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return transport.RoundTrip(req)
}

func generateToken(privateKeyStr, subscriber, origin string) (string, error) {
	privBytes, err := base64.RawURLEncoding.DecodeString(privateKeyStr)
	if err != nil {
		return "", err
	}
	
	curve := elliptic.P256()
	privKey := new(ecdsa.PrivateKey)
	privKey.Curve = curve
	privKey.D = new(big.Int).SetBytes(privBytes)
	privKey.PublicKey.X, privKey.PublicKey.Y = curve.ScalarBaseMult(privBytes)

	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"aud": origin,
		"exp": time.Now().Add(45 * time.Minute).Unix(), // 45 min to be safe (<1h)
		"sub": subscriber,
	})

	return token.SignedString(privKey)
}
	
func (s *Service) sendToApple(endpoint, p256dh, auth string, payload []byte) error {
	sub := &webpush.Subscription{
		Endpoint: endpoint,
		Keys: webpush.Keys{
			P256dh: p256dh,
			Auth:   auth,
		},
	}
	
	subscriber := os.Getenv("VAPID_SUBJECT")
	if subscriber == "" {
		// Use a safe default for Apple? or just the generic one
		subscriber = "mailto:admin@pushem.local"
	}
	if !strings.HasPrefix(subscriber, "mailto:") {
		subscriber = "mailto:" + subscriber
	}

	// Use custom transport to intercept and fix VAPID header
	client := &http.Client{
		Transport: &AppleTransport{
			PrivateKey: s.privateKey,
			PublicKey:  s.publicKey,
			Subscriber: subscriber,
		},
		Timeout: 30 * time.Second,
	}

	resp, err := webpush.SendNotification(payloadBytes, sub, &webpush.Options{
		// We still pass keys here so the library effectively "works", 
		// but our Transport will OVERWRITE the Authorization header.
		Subscriber:      subscriber,
		VAPIDPrivateKey: s.privateKey,
		VAPIDPublicKey:  s.publicKey,
		TTL:             86400,
		Urgency:         webpush.UrgencyHigh,
		HTTPClient:      client, 
	})
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 410 {
		return fmt.Errorf("subscription expired (410 Gone)")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Push service error response: %s", string(body))
		return fmt.Errorf("push service returned status: %d", resp.StatusCode)
	}

	return nil
}
