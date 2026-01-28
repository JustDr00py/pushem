package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"pushem/internal/api"
	"pushem/internal/db"
	"pushem/internal/webpush"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func startMessageCleanup(database *db.DB) {
	// Get configuration from environment variables
	retentionDays := 7 // Default: keep messages for 7 days
	if days := os.Getenv("MESSAGE_RETENTION_DAYS"); days != "" {
		if parsed, err := strconv.Atoi(days); err == nil && parsed > 0 {
			retentionDays = parsed
		}
	}

	cleanupInterval := 24 * time.Hour // Default: run cleanup once per day
	if hours := os.Getenv("CLEANUP_INTERVAL_HOURS"); hours != "" {
		if parsed, err := strconv.Atoi(hours); err == nil && parsed > 0 {
			cleanupInterval = time.Duration(parsed) * time.Hour
		}
	}

	log.Printf("Message cleanup: retention=%d days, interval=%v", retentionDays, cleanupInterval)

	// Run cleanup in background
	go func() {
		// Run initial cleanup after 1 minute
		time.Sleep(1 * time.Minute)

		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for {
			count, err := database.DeleteOldMessages(retentionDays)
			if err != nil {
				log.Printf("Error during message cleanup: %v", err)
			} else if count > 0 {
				log.Printf("Cleaned up %d old messages (older than %d days)", count, retentionDays)

				// Log current message count
				if total, err := database.GetMessageCount(); err == nil {
					log.Printf("Current message count: %d", total)
				}
			}

			<-ticker.C
		}
	}()
}

func main() {
	log.Println("Starting Pushem Server...")

	database, err := db.New("pushem.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()
	log.Println("Database initialized")

	webpushService, err := webpush.NewService()
	if err != nil {
		log.Fatalf("Failed to initialize webpush service: %v", err)
	}
	log.Println("Web Push service initialized")

	// Start message cleanup goroutine
	startMessageCleanup(database)

	handler := api.NewHandler(database, webpushService)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/vapid-public-key", handler.GetVAPIDPublicKey)
	r.Post("/subscribe/{topic}", handler.Subscribe)
	r.Post("/publish/{topic}", handler.Publish)
	r.Get("/history/{topic}", handler.GetHistory)
	r.Delete("/history/{topic}", handler.ClearHistory)
	r.Post("/topics/{topic}/protect", handler.ProtectTopic)

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "web/dist"
	}
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		log.Printf("Warning: Frontend directory '%s' not found. Frontend will not be available.", staticDir)
	} else {
		fileServer := http.FileServer(http.Dir(staticDir))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix("/", fileServer).ServeHTTP(w, r)
		})
		log.Printf("Serving frontend from '%s'", staticDir)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on :%s", port)
	log.Printf("API endpoints:")
	log.Printf("  GET  /vapid-public-key")
	log.Printf("  POST /subscribe/{topic}")
	log.Printf("  POST /publish/{topic}")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
