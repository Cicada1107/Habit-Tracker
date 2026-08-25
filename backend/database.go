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
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		habit_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		duration_minutes INTEGER NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(habit_id) REFERENCES habits(id) ON DELETE CASCADE,
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
		INSERT INTO events (id, habit_id, user_id, start_time, end_time, duration_minutes)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			start_time = excluded.start_time,
			end_time = excluded.end_time,
			duration_minutes = excluded.duration_minutes;
	`

	// Temporarily saving 'title' as habit_id for now, since we don't have a habit_id yet. In the future, we can map events to habits.
	_, err := db.Exec(query, eventID, title, userID, startTime, endTime, durationMinutes)
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

	rows, err := db.Query("SELECT habit_id, start_time, duration_minutes FROM events WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []EventResponse
	for rows.Next() {
		var habitID string
		var startTime time.Time
		var durationMinutes int
		if err := rows.Scan(&habitID, &startTime, &durationMinutes); err != nil {
			return nil, err
		}

		event := EventResponse{
			Title:    habitID,
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
