package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Habit struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateHabitRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func handleGetHabits(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	db := InitDB()
	defer db.Close()

	rows, err := db.Query("SELECT id, user_id, name, description, created_at FROM habits WHERE user_id = ?", userID)
	if err != nil {
		http.Error(w, "Failed to fetch habits", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var habits []Habit
	for rows.Next() {
		var h Habit
		var desc sql.NullString
		if err := rows.Scan(&h.ID, &h.UserID, &h.Name, &desc, &h.CreatedAt); err != nil {
			continue
		}
		if desc.Valid {
			h.Description = desc.String
		}
		habits = append(habits, h)
	}

	w.Header().Set("Content-Type", "application/json")
	// Always return an array (even if empty) to the frontend
	if habits == nil {
		habits = []Habit{}
	}
	json.NewEncoder(w).Encode(habits)
}

func handleCreateHabit(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	var req CreateHabitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	db := InitDB()
	defer db.Close()

	habitID := uuid.New().String()
	query := `INSERT INTO habits (id, user_id, name, description) VALUES (?, ?, ?, ?)`
	
	_, err := db.Exec(query, habitID, userID, req.Name, req.Description)
	if err != nil {
		http.Error(w, "Failed to create habit", http.StatusInternalServerError)
		return
	}

	// Make sure past IGNORED events get re-evaluated now that a new habit exists!
	db.Exec("UPDATE events SET mapping_status = 'PENDING' WHERE user_id = ? AND mapping_status = 'IGNORED'", userID)
	
	// Fetch google_id to trigger the mapper
	var googleID string
	if err := db.QueryRow("SELECT google_id FROM users WHERE id = ?", userID).Scan(&googleID); err == nil {
		go RunAICategorizer(db, googleID)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": habitID, "status": "success"})
}
