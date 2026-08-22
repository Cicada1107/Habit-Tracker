package main

import (
	"fmt"
	"net/http"
)

func handleGoogleWebhook(w http.ResponseWriter, r *http.Request) {
	// Google sends useful info in the headers
	resourceState := r.Header.Get("X-Goog-Resource-State")

	if resourceState == "sync" {
		fmt.Println("Google Webhook Connected.")
		w.WriteHeader(http.StatusOK)
		return
	}

	fmt.Println("[Alert] Webhook Received")

	var googleID string
	db := InitDB()
	defer db.Close()

	// grab the google id of the user we logged in as earlier
	err := db.QueryRow("SELECT google_id FROM users LIMIT 1").Scan(&googleID)

	if err == nil && googleID != "" {
		EnqueueSyncJob(googleID)
	}

	w.WriteHeader(http.StatusOK)
}
