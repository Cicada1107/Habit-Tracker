package main

import (
	"encoding/json"
	"net/http"
)

// EventResponse represents the structure of a Google Calendar event response
type EventResponse struct {
	Title    string `json:"title"`
	Start    string `json:"start"`
	Duration string `json:"end"`
}

func handleGetEvents(w http.ResponseWriter, r *http.Request) {
	db := InitDB()
	defer db.Close()

	// 1. Get the user_id from the context (set by AuthMiddleware)
	userID := r.Context().Value("user_id").(string)

	// 2. Fetch the user's Google ID from the database
	var googleID string
	err := db.QueryRow("SELECT google_id FROM users WHERE id = ?", userID).Scan(&googleID)
	if err != nil {
		http.Error(w, "Failed to fetch Google ID", http.StatusInternalServerError)
		return
	}

	// 3. Fetch events from Google Calendar using the Google ID
	events, err := fetchCalendarEventsFromDB(googleID)
	if err != nil {
		http.Error(w, "Failed to fetch events from Google Calendar", http.StatusInternalServerError)
		return
	}

	// 4. Respond with the events in JSON format
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
