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

// GetEventsInRange retrieves events for any time frame for a user from the database
func GetEventsInRange(db *sql.DB, userID string, startTime string, endTime string) ([]EventSummary, error) {
	query := `
		SELECT COALESCE(h.name, e.raw_title), e.duration_minutes, e.start_time 
		FROM events e
		LEFT JOIN habits h ON e.habit_id = h.id
		WHERE e.user_id = ? AND datetime(e.start_time) >= datetime(?) AND datetime(e.start_time) <= datetime(?) 
		ORDER BY e.start_time ASC`
		
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

// CalculateHabitProbability predicts adherence probability for a tracked habit based on past data.
func CalculateHabitProbability(db *sql.DB, userID string, habitName string, startTime string, endTime string) (ProbabilityStats, error) {
	// Notice we only query mapped habits here!
	query := `
		SELECT e.start_time 
		FROM events e
		JOIN habits h ON e.habit_id = h.id
		WHERE e.user_id = ? AND h.name LIKE ? AND datetime(e.start_time) >= datetime(?) AND datetime(e.start_time) <= datetime(?) 
		ORDER BY e.start_time ASC`

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

	if len(dates) < 2 {
		stats.PredictedProbability = 0.0
		stats.ConsistencyScore = 0.0
		return stats, nil
	}

	var gaps []float64
	var sumGaps float64
	for i := 1; i < len(dates); i++ {
		gap := dates[i].Sub(dates[i-1]).Hours()
		gaps = append(gaps, gap)
		sumGaps += gap
	}

	meanGap := sumGaps / float64(len(gaps))

	var sumVariance float64
	for _, gap := range gaps {
		sumVariance += math.Pow(gap-meanGap, 2)
	}
	variance := sumVariance / float64(len(gaps))
	stdDev := math.Sqrt(variance)

	consistency := (meanGap / (meanGap + stdDev)) * 100 
	if math.IsNaN(consistency) {
		consistency = 50.0 
	}
	stats.ConsistencyScore = math.Round(consistency*100) / 100 

	hoursSinceLast := time.Now().Sub(dates[len(dates)-1]).Hours()
	zScore := math.Abs(hoursSinceLast-meanGap) / (stdDev + 1)

	probability := math.Max(0, 100-(zScore*25))
	finalProb := (probability * 0.7) + (stats.ConsistencyScore * 0.3)
	stats.PredictedProbability = math.Round(finalProb*100) / 100 

	return stats, nil
}
