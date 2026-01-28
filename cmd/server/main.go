package main

import (
	"log"
	"net/http"
	"os"

	"pushem/internal/api"
	"pushem/internal/db"
	"pushem/internal/webpush"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

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
