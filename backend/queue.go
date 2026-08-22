package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client
var ctx = context.Background()

func InitRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("Could not connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")
}

// EnqueueSyncJob adds a user's Google ID to the queue to be processed later
func EnqueueSyncJob(googleId string) {
	err := rdb.LPush(ctx, "sync_queue", googleId).Err()
	if err != nil {
		fmt.Printf("failed to enqueue sync job: %v", err)
		return
	}

	fmt.Printf("Queued sync job for the user: %s/n", googleId)
}

// StartWorker runs in the background and processes jobs from the queue
func StartWorker(db *sql.DB) {
	log.Println("Background worker started. Listening for jobs...")

	// Continuously listen for jobs in the queue
	for {
		result, err := rdb.BRPop(ctx, 0, "sync_queue").Result()
		if err != nil {
			log.Printf("Worker Error: %v", err)
			continue
		}

		// result[0] is the queue name, result[1] is the Google ID
		googleId := result[1]
		log.Printf("Processing sync job for user: %s", googleId)

		// 1. Fetch the user's access token from the database
		accessToken, err := GetAccessToken(db, googleId)
		if err == nil {
			FetchCalendarEvents(db, accessToken, googleId)
			fmt.Println("Worker finished the job successfully")
		} else {
			log.Printf("Failed to fetch access token for user %s: %v", googleId, err)
		}
	}
}
