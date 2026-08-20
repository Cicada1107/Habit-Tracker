package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	defer db.Close()

	// 2. Set up the router
	r := chi.NewRouter()
	r.Use(middleware.Logger)    //Logs every api request to terminal
	r.Use(middleware.Recoverer) //Prevents the server from crashing if there's an error

	// 3. Define the routes

	// Health Route
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Habit Coach API is up and running!"))
	})

	// Login route
	r.Get("/auth/google/login", handleGoogleLogin)

	// Callback route
	r.Get("/auth/google/callback", handleGoogleCallback)

	// 4. Start the server
	port := "8080"
	log.Printf("Starting server on :%s", port)
	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
