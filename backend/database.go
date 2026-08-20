package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func initDB() *sql.DB {
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
		created_at DATETIME DEFAULT CURRENT TIMESTAMP
	);


	CREATE TABLE IF NOT EXISTS habits(
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		habit_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		duration_minutes INTEGER NOT NULL,
		timestamp DATETIME DEFAULT CURRENT TIMESTAMP,
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
