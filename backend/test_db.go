package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./habit_coach.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT habit_id, duration_minutes FROM events")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var title string
		var dur int
		rows.Scan(&title, &dur)
		fmt.Printf("Event: %s | Duration: %d\n", title, dur)
		count++
	}
	fmt.Printf("Total events: %d\n", count)
}
