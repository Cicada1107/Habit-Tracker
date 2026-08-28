package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/genai"
)

type AIMappingResult struct {
	Mappings []struct {
		EventID string  `json:"event_id"`
		HabitID *string `json:"habit_id"`
	} `json:"mappings"`
}

// RunAICategorizer grabs PENDING events and asks Gemini to map them to user's defined habits.
func RunAICategorizer(db *sql.DB, googleID string) {
	ctx := context.Background()

	// 1. Get userID
	var userID string
	err := db.QueryRow("SELECT id FROM users WHERE google_id = ?", googleID).Scan(&userID)
	if err != nil {
		log.Printf("Mapper: Failed to find user: %v", err)
		return
	}

	// 2. Fetch User Habits
	rows, err := db.Query("SELECT id, name, description FROM habits WHERE user_id = ?", userID)
	if err != nil {
		log.Printf("Mapper: Failed to fetch habits: %v", err)
		return
	}
	defer rows.Close()

	var habits []map[string]string
	for rows.Next() {
		var id, name string
		var desc sql.NullString
		if err := rows.Scan(&id, &name, &desc); err == nil {
			habit := map[string]string{"id": id, "name": name, "description": ""}
			if desc.Valid {
				habit["description"] = desc.String
			}
			habits = append(habits, habit)
		}
	}
	
	if len(habits) == 0 {
		log.Println("Mapper: No habits defined by user. Skipping categorization.")
		// We still need to mark pending events as IGNORED so they don't pile up forever
		db.Exec("UPDATE events SET mapping_status = 'IGNORED' WHERE user_id = ? AND mapping_status = 'PENDING'", userID)
		return
	}

	// 3. Fetch PENDING events
	eventRows, err := db.Query("SELECT id, raw_title FROM events WHERE user_id = ? AND mapping_status = 'PENDING' LIMIT 50", userID)
	if err != nil {
		log.Printf("Mapper: Failed to fetch events: %v", err)
		return
	}
	defer eventRows.Close()

	var pendingEvents []map[string]string
	for eventRows.Next() {
		var id, title string
		if err := eventRows.Scan(&id, &title); err == nil {
			pendingEvents = append(pendingEvents, map[string]string{"id": id, "title": title})
		}
	}

	if len(pendingEvents) == 0 {
		log.Println("Mapper: No pending events to categorize.")
		return
	}

	// 4. Construct Prompt
	habitsJSON, _ := json.Marshal(habits)
	eventsJSON, _ := json.Marshal(pendingEvents)
	
	prompt := fmt.Sprintf(`You are an AI data mapper for a Habit Tracking app.
Your job is to map raw Google Calendar event titles to the user's defined tracking goals (Habits).
Be smart about semantics. "Hit the weights" maps to "Gym", "Dentist" maps to nothing.

User's Defined Habits:
%s

Unmapped Calendar Events:
%s

You must output a JSON object with a single key "mappings", containing an array of objects.
Each object must have "event_id" (string) and "habit_id" (string or null).
If an event clearly relates to a habit, map it. If it doesn't relate to ANY of the habits, set habit_id to null.`, string(habitsJSON), string(eventsJSON))

	// 5. Query Gemini
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Printf("Mapper: Failed to initialize AI client: %v", err)
		return
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		Temperature:      func(f float32) *float32 { return &f }(0.1),
	}
	
	model := "gemini-3.5-flash-lite"
	resp, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), config)
	if err != nil {
		log.Printf("Mapper: Gemini API call failed: %v", err)
		return
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Printf("Mapper: Invalid Gemini response")
		return
	}

	jsonString := resp.Candidates[0].Content.Parts[0].Text

	type Mapping struct {
		EventID string  `json:"event_id"`
		HabitID *string `json:"habit_id"`
	}
	
	var mappings []Mapping
	
	// Try parsing as array directly (which flash-lite prefers to output)
	if err := json.Unmarshal([]byte(jsonString), &mappings); err != nil {
		// Fallback: try parsing as object with "mappings" key
		var obj struct {
			Mappings []Mapping `json:"mappings"`
		}
		if err2 := json.Unmarshal([]byte(jsonString), &obj); err2 != nil {
			log.Printf("Mapper: Failed to parse JSON response: %v\nRaw Output: %s", err, jsonString)
			return
		}
		mappings = obj.Mappings
	}

	// 6. Update Database
	log.Printf("Mapper: Successfully mapped %d events", len(mappings))
	for _, m := range mappings {
		if m.HabitID != nil && *m.HabitID != "" {
			db.Exec("UPDATE events SET habit_id = ?, mapping_status = 'MAPPED' WHERE id = ?", *m.HabitID, m.EventID)
		} else {
			db.Exec("UPDATE events SET habit_id = NULL, mapping_status = 'IGNORED' WHERE id = ?", m.EventID)
		}
	}
}
