package main

import (
	"database/sql"
	"math"
	"time"
)

// --------------------------------------------
// DATA RETREIVAL TOOLS
// --------------------------------------------

type EventSummary struct {
	HabitName string `json:"habit_name"`
	Duration  int    `json:"duration_minutes"`
	Date      string `json:"date"`
}

// GetEvetnsInRange retreives events for any time frame (past/present/future) for a user from the database
func GetEventsInRange(db *sql.DB, userID string, startTime string, endTime string) ([]EventSummary, error) {
	query := `SELECT habit_id, duration_minutes, start_time FROM events WHERE user_id = ? AND start_time >= ? AND start_time <= ? ORDER BY start_time ASC`
	rows, err := db.Query(query, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []EventSummary
	for rows.Next() {
		var e EventSummary
		rows.Scan(&e.HabitName, &e.Duration, &e.Date)
		events = append(events, e)
	}
	return events, nil
}

// --------------------------------------------
// STATISTICAL DATA ANALYSIS TOOLS
// --------------------------------------------

type ProbabilityStats struct {
	HabitName            string  `json:"habit_name"`
	TotalOccurances      int     `json:"total_occurances"`
	ConsistencyScore     float64 `json:"consistency_score_percentage"`
	PredictedProbability float64 `json:"predicted_probability_percentage"`
}

// CalculateHabitProbability is a statistical function that predicts adherence probability for a habit based on past data. It returns a ProbabilityStats struct with the results.
func CalculateHabitProbability(db *sql.DB, userID string, habitName string, startTime string, endTime string) (ProbabilityStats, error) {
	query := `SELECT start_time FROM EVENTS WHERE user_id = ? AND habit_id LIKE ? AND start_time >= ? AND start_time <= ? ORDER BY start_time ASC`

	rows, err := db.Query(query, userID, "%"+habitName+"%", startTime, endTime)
	if err != nil {
		return ProbabilityStats{}, err
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var dateStr string
		rows.Scan(&dateStr)
		if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
			dates = append(dates, t)
		}
	}

	stats := ProbabilityStats{
		HabitName:       habitName,
		TotalOccurances: len(dates),
	}

	// If less than 2 events, not enough data to calculate statistics
	if len(dates) < 2 {
		stats.PredictedProbability = 0.0
		stats.ConsistencyScore = 0.0
		return stats, nil
	}

	// 1. Caculate gaps (in hours) between each time the habit wasw performed
	var gaps []float64
	var sumGaps float64
	for i := 1; i < len(dates); i++ {
		gap := dates[i].Sub(dates[i-1]).Hours()
		gaps = append(gaps, gap)
		sumGaps += gap
	}

	// 2. Calculate average gap
	meanGap := sumGaps / float64(len(gaps))

	// 3. Calculate Variance and Standard Deviation (How consistent?)
	var sumVariance float64
	for _, gap := range gaps {
		sumVariance += math.Pow(gap-meanGap, 2)
	}
	variance := sumVariance / float64(len(gaps))
	stdDev := math.Sqrt(variance)

	consistency := (meanGap / (meanGap + stdDev)) * 100 // Higher score means more consistent
	if math.IsNaN(consistency) {
		consistency = 50.0 // Default to 50% if NaN
	}
	stats.ConsistencyScore = math.Round(consistency*100) / 100 // Round to 2 decimal places

	// 4. Calc Predicted probability of doing it today
	hoursSinceLast := time.Now().Sub(dates[len(dates)-1]).Hours()
	zScore := math.Abs(hoursSinceLast-meanGap) / (stdDev + 1)

	probability := math.Max(0, 100-(zScore*25))
	finalProb := (probability * 0.7) + (stats.ConsistencyScore * 0.3) // Weighted average of probability and consistency
	stats.PredictedProbability = math.Round(finalProb*100) / 100      // Round to 2 decimal places

	return stats, nil
}
