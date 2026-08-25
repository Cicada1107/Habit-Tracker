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

	model := client.GenerativeModel("gemini-1.5-flash")

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
	// Injecting current time for the LLM to be able to have context about the point in time
	currentTime := time.Now().Format(time.RFC3339)
	systemPrompt := fmt.Sprintf(`You are the 'Habit Coach' AI. The current time is %s. You have access to tools to fetch user data and calculate probabilities. ALWAYS use your tools to check the user's schedule or stats before answering questions about their habits. Keep your final answer concise and encouraging, formatted in markdown. Be unbiased, and try to be to the point and clear.`, currentTime)

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	session := model.StartChat()
	resp, err := session.SendMessage(ctx, genai.Text(req.Question))
	if err != nil {
		http.Error(w, "Failed to get response from AI", http.StatusInternalServerError)
		return
	}

	var answer string

	// 2. the ReAct loop (Max 3 iterations to prevent infinite loops)
	for i := 0; i < 3; i++ {
		toolCalled := false

		for _, part := range resp.Candidates[0].Content.Parts {
			if funcCall, ok := part.(genai.FunctionCall); ok {
				toolCalled = true
				fmt.Println("Calling tool: ", funcCall.Name)
				var toolResponse map[string]any

				// 3. Route the tool call to the appropriate function
				switch funcCall.Name {

				case "get_events_in_range":
					startTime := funcCall.Args["start_time"].(string)
					endTime := funcCall.Args["end_time"].(string)
					events, err := GetEventsInRange(db, userID, startTime, endTime)
					if err != nil {
						http.Error(w, "Error fetching events", http.StatusInternalServerError)
						return
					}
					toolResponse = map[string]any{"events": events}

				case "calculate_habit_probability":
					habit := funcCall.Args["habit_name"].(string)
					startTime := funcCall.Args["start_time"].(string)
					endTime := funcCall.Args["end_time"].(string)
					stats, err := CalculateHabitProbability(db, userID, habit, startTime, endTime)
					if err != nil {
						http.Error(w, "Error calculating habit probability", http.StatusInternalServerError)
						return
					}
					toolResponse = map[string]any{"stats": stats}
				}

				// 4. Feed the tool response back to the model
				resp, err = session.SendMessage(ctx, genai.FunctionResponse{
					Name:     funcCall.Name,
					Response: toolResponse,
				})
				if err != nil {
					http.Error(w, "Failed to get response from AI after tool call", http.StatusInternalServerError)
					return
				}
			} else if text, ok := part.(genai.Text); ok {
				answer = string(text)
			}
		}

		// If no tool was called in this iteration, the AI has formulated its final answer! Exit the ReAct loop.
		if !toolCalled {
			break
		}
	}

	// 5. Send the final answer to the frontend (MUST be outside the for loop!)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChartResponse{Answer: answer})
}
