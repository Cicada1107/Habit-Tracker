package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// FetchCalendarEvents fetches the time-based blocks from user's google calendar
func FetchCalendarEvents(db *sql.DB, accessToken string, googleID string) {
	ctx := context.Background()
	token := &oauth2.Token{AccessToken: accessToken}
	client := googleOauthConfig.Client(ctx, token)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Printf("Unable to retrieve Calendar client: %v", err)
		return
	}

	timeMin := time.Now().AddDate(0, 0, -7).Format(time.RFC3339) // Fetch events from the last 7 days
	events, err := srv.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(timeMin).MaxResults(50).OrderBy("startTime").Do()
	if err != nil {
		log.Printf("Unable to retrieve user's events: %v", err)
		return
	}

	var userId string
	err = db.QueryRow("SELECT id FROM users WHERE google_id = ?", googleID).Scan(&userId)
	if err != nil {
		log.Printf("Unable to find user in DB: %v", err)
		return
	}

	fmt.Println("Saving events to DB")
	for _, item := range events.Items {
		// Ignore holdiays etc by only considering events with a start and end time
		if item.Start.DateTime == "" || item.End.DateTime == "" {
			continue
		}

		startTime, err := time.Parse(time.RFC3339, item.Start.DateTime)
		endTime, err := time.Parse(time.RFC3339, item.End.DateTime)
		if err != nil {
			log.Printf("Error parsing event time: %v", err)
			continue
		}

		err = SaveEvent(db, item.Id, userId, item.Summary, startTime, endTime)
		if err != nil {
			log.Printf("Error saving event to DB: %v", err)
		} else {
			log.Printf("Event saved successfully: %s", item.Summary)
		}
	}
}
