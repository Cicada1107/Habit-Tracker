package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env variables
	if err := godotenv.Load(); err != nil {
		log.Println("Environment Variables not set")
	}

	initOAuth()

	// 1. Initialize the SQLite database
	db := InitDB()
	InitRedis()
	go StartWorker(db)
	defer db.Close()

	// 2. Set up the router
	r := chi.NewRouter()
	r.Use(middleware.Logger)    //Logs every api request to terminal
	r.Use(middleware.Recoverer) //Prevents the server from crashing if there's an error
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
	}))

	// 3. Define the routes

	// Health Routes
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Habit Coach API is up and running!"))
	})

	// Login, consent & redirect routes
	r.Get("/auth/google/login", handleGoogleLogin)
	r.Get("/auth/google/callback", handleGoogleCallback)
	r.Get("/api/events", AuthMiddleware(handleGetEvents))

	// Fetching user data routes

	// Webhook routes
	r.Post("/api/webhooks/google", handleGoogleWebhook)

	// 4. Start the server
	port := "8080"
	log.Printf("Starting server on :%s", port)
	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
