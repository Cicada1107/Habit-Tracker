package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type ChatRequest struct {
	Question string `json:"question"`
}

type ChartResponse struct {
	Answer string `json:"answer"`
}

// handlehabitCoachChat is the agentic ai endpoint that takes a question and returns an answer
func handleHabitCoachChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(string)
	db := InitDB()
	defer db.Close()

	ctx := r.Context()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		http.Error(w, "Failed to connect to AI service", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-3.5-flash")

	// 1. DEFINE THE TOOLBOX
	toolbox := &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "get_events_in_range",
				Description: "Retrieve events for a user within a specified time range. Use this to check past behaviour or future schedules. start_time and end_time must be in RFC3339 string format.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"start_time": {Type: genai.TypeString, Description: "RFC3339 formatted start time for the range."},
						"end_time":   {Type: genai.TypeString, Description: "RFC3339 formatted end time for the range."},
					},
					Required: []string{"start_time", "end_time"},
				},
			},
			{
				Name:        "calculate_habit_probability",
				Description: "Calculates the statistical probability of the user adhering to a specific habit, along with their consistency score. Use this to predict future adherence based on past behaviour.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"habit_name": {Type: genai.TypeString, Description: "The name of the habit to calculate probability for."},
						"start_time": {Type: genai.TypeString, Description: "RFC3339 formatted start time for the range."},
						"end_time":   {Type: genai.TypeString, Description: "RFC3339 formatted end time for the range."},
					},
					Required: []string{"habit_name", "start_time", "end_time"},
				},
			},
		},
	}

	model.Tools = append(model.Tools, toolbox)
	currentTime := time.Now().Format(time.RFC3339)
	systemPrompt := fmt.Sprintf(`You are the 'Habit Coach' AI. The current time is %s. You have access to tools to fetch user data and calculate probabilities. ALWAYS use your tools to check the user's schedule or stats before answering questions about their habits. Keep your final answer concise and encouraging, formatted in markdown. Be unbiased, and try to be to the point and clear.`, currentTime)

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	var answer string
	currentPrompt := req.Question

	// 2. The STATELESS ReAct Loop (Max 3 iterations)
	// Bypasses both the Quota limit of 2.5 and the thought_signature bug of 3.5!
	for i := 0; i < 3; i++ {
		toolCalled := false
		resp, err := model.GenerateContent(ctx, genai.Text(currentPrompt))
		
		if err != nil {
			fmt.Printf("🔴 AI ERROR: %v\n", err)
			http.Error(w, "AI generation failed", http.StatusInternalServerError)
			return
		}

		for _, part := range resp.Candidates[0].Content.Parts {
			if funcCall, ok := part.(genai.FunctionCall); ok {
				toolCalled = true
				fmt.Println("🤖 AI is calling tool:", funcCall.Name)

				var toolData any

				if funcCall.Name == "get_events_in_range" {
					startTime := funcCall.Args["start_time"].(string)
					endTime := funcCall.Args["end_time"].(string)
					events, _ := GetEventsInRange(db, userID, startTime, endTime)
					toolData = events
					
				} else if funcCall.Name == "calculate_habit_probability" {
					habit := funcCall.Args["habit_name"].(string)
					startTime := funcCall.Args["start_time"].(string)
					endTime := funcCall.Args["end_time"].(string)
					stats, _ := CalculateHabitProbability(db, userID, habit, startTime, endTime)
					toolData = stats
				}

				// Convert data to JSON string
				dataBytes, _ := json.MarshalIndent(toolData, "", "  ")

				// Append the tool results to the prompt for the next iteration!
				currentPrompt += fmt.Sprintf("\n\nSystem Note: You called tool '%s' which returned:\n%s\nFormulate your final answer, or call another tool if you need more data.", funcCall.Name, string(dataBytes))
				break // Break out of the parts loop, let the ReAct loop iterate

			} else if text, ok := part.(genai.Text); ok {
				answer += string(text)
			}
		}

		if !toolCalled {
			break
		}
	}

	// 5. Send the final answer to the frontend (MUST be outside the for loop!)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChartResponse{Answer: answer})
}
