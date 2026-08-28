package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./habit_coach.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	createTables(db)
	return db
}

func createTables(db *sql.DB) {
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		google_id TEXT UNIQUE NOT NULL,
		email TEXT NOT NULL,
		access_token TEXT,
		refresh_token TEXT,
		token_expiry DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS habits(
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		raw_title TEXT NOT NULL,
		habit_id TEXT,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		duration_minutes INTEGER NOT NULL,
		mapping_status TEXT DEFAULT 'PENDING',
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(habit_id) REFERENCES habits(id) ON DELETE SET NULL,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`

	_, err := db.Exec(createTablesSQL)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
	log.Println("Database tables initialized successfully!")
}

// Inserts or updates a user in the database
func SaveUser(db *sql.DB, googleID, email, accessToken, refreshToken string, tokenExpiry time.Time) error {
	id := uuid.New().String()

	query := `
		INSERT INTO users (id, google_id, email, access_token, refresh_token, token_expiry)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(google_id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			token_expiry = excluded.token_expiry;
	`

	_, err := db.Exec(query, id, googleID, email, accessToken, refreshToken, tokenExpiry)
	return err
}

// Inserts an event into the database
func SaveEvent(db *sql.DB, eventID, userID, title string, startTime, endTime time.Time) error {
	durationMinutes := int(endTime.Sub(startTime).Minutes())

	query := `
		INSERT INTO events (id, user_id, raw_title, start_time, end_time, duration_minutes, mapping_status)
		VALUES (?, ?, ?, ?, ?, ?, 'PENDING')
		ON CONFLICT(id) DO UPDATE SET
			raw_title = excluded.raw_title,
			start_time = excluded.start_time,
			end_time = excluded.end_time,
			duration_minutes = excluded.duration_minutes,
			mapping_status = 'PENDING'; 
	`

	// On update, we reset mapping_status to PENDING in case the title changed and needs re-evaluating by AI
	_, err := db.Exec(query, eventID, userID, title, startTime, endTime, durationMinutes)
	return err
}

// Fetches a user's access token from the database using their Google ID
func GetToken(db *sql.DB, googleID string) (string, string, time.Time, error) {
	var accessToken, refreshToken string
	var expiry time.Time
	query := `SELECT access_token, refresh_token, token_expiry FROM users WHERE google_id = ?`
	err := db.QueryRow(query, googleID).Scan(&accessToken, &refreshToken, &expiry)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return accessToken, refreshToken, expiry, nil
}

// Fetches all of a user's calendar evetns from the DB, given their google ID
func fetchCalendarEventsFromDB(googleID string) ([]EventResponse, error) {
	db := InitDB()
	defer db.Close()

	var userID string
	err := db.QueryRow("SELECT id FROM users WHERE google_id = ?", googleID).Scan(&userID)
	if err != nil {
		return nil, err
	}

	// For now, return raw_title as Title to keep the frontend working during transition.
	// We also optionally fetch the mapped habit name if it exists.
	query := `
		SELECT e.raw_title, e.start_time, e.duration_minutes, h.name 
		FROM events e LEFT JOIN habits h 
		ON e.habit_id = h.id
		WHERE e.user_id = ?
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []EventResponse
	for rows.Next() {
		var rawTitle string
		var startTime time.Time
		var durationMinutes int
		var habitName sql.NullString

		if err := rows.Scan(&rawTitle, &startTime, &durationMinutes, &habitName); err != nil {
			return nil, err
		}

		// Use the resolved habit name if mapped, otherwise fallback to raw title for now
		finalTitle := rawTitle
		if habitName.Valid {
			finalTitle = habitName.String
		}

		event := EventResponse{
			Title:    finalTitle,
			Start:    startTime.Format(time.RFC3339),
			Duration: durationMinutes,
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}
