package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/genai"
)

type ChatRequest struct {
	Question string `json:"question"`
}

type ChatResponse struct {
	Answer  string `json:"answer"`
	Thought string `json:"thought"`
}

func handleHabitCoachChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	json.NewDecoder(r.Body).Decode(&req)

	userID := r.Context().Value("user_id").(string)
	db := InitDB()
	defer db.Close()
	ctx := r.Context()

	// 1. Initialize the AI client
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		http.Error(w, "Failed to initialize AI client", http.StatusInternalServerError)
		return
	}

	// 2. Define the tool box
	toolbox := &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "get_events_in_range",
				Description: "Fetches a user's events from the database for a given time range.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"start_time": {Type: genai.TypeString, Description: "Start time in RFC3339 format"},
						"end_time":   {Type: genai.TypeString, Description: "End time in RFC3339 format"},
					},
					Required: []string{"start_time", "end_time"},
				},
			},
			{
				Name:        "calculate_habit_probability",
				Description: "Calculates the probability of a user adhering to a habit based on past data.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"habit_name": {Type: genai.TypeString, Description: "Name of the habit"},
						"start_time": {Type: genai.TypeString, Description: "Start time in RFC3339 format"},
						"end_time":   {Type: genai.TypeString, Description: "End time in RFC3339 format"},
					},
					Required: []string{"habit_name", "start_time", "end_time"},
				},
			},
		},
	}

	currentTime := time.Now().Format(time.RFC3339)
	systemPrompt := fmt.Sprintf(`You are the 'Habit Coach' AI. The current time is %s. You have access to the following tools:
		1. get_events_in_range: Fetches a user's events from the database for a given time range.
		2. calculate_habit_probability: Calculates the probability of a user adhering to a habit based on past data.

		Use these tools to provide accurate and helpful responses to the user's questions about their habits and events.
		ALWAYS check stats before answering. Be unbiased, concise, professional and to the point.`, currentTime)

	// 3. Create the Configuration
	config := &genai.GenerateContentConfig{
		Tools: []*genai.Tool{toolbox},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{genai.NewPartFromText(systemPrompt)},
		},
	}

	// 4. Start the Chat Session
	session, _ := client.Chats.Create(ctx, "gemini-3.5-flash-lite", config, nil)

	// send the initial user message
	resp, err := session.SendMessage(ctx, *genai.NewPartFromText(req.Question))
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send message to AI: %v", err)
		fmt.Println("🔴", errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	var answer string

	// 5. Tool ReAct loop
	for i := 0; i < 10; i++ {
		toolCalled := false

		var funcResponses []genai.Part

		for _, part := range resp.Candidates[0].Content.Parts {
			
			// If tools have been called
			if part.FunctionCall != nil {
				toolCalled = true
				funcCall := part.FunctionCall
				fmt.Println("Calling tool: ", funcCall.Name)
				
				var toolData any
				switch funcCall.Name {
					case "get_events_in_range":
						start := funcCall.Args["start_time"].(string)
						end := funcCall.Args["end_time"].(string)
						events, _ := GetEventsInRange(db, userID, start, end)
						toolData = events
					case "calculate_habit_probability":
						habit := funcCall.Args["habit_name"].(string)
						start := funcCall.Args["start_time"].(string)
						end := funcCall.Args["end_time"].(string)
						stats, _ := CalculateHabitProbability(db, userID, habit, start, end)
						toolData = stats
					default:
						toolData = map[string]string{"error": "Unknown tool called"}
				}
				
				// JSON marshalling to protect SDK
				toolDataWrapped := map[string]any{"result": toolData}
				b, _ := json.Marshal(toolDataWrapped)
				var cleanData map[string]any
				json.Unmarshal(b, &cleanData)

				funcResponses = append(funcResponses, *genai.NewPartFromFunctionResponse(funcCall.Name, cleanData))
			} else if part.Text != "" {
				answer += part.Text
			}
		}

		if len(funcResponses) > 0 {
			resp, err = session.SendMessage(ctx, funcResponses...)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to send tool response to AI: %v", err)
				fmt.Println("🔴", errMsg)
				http.Error(w, errMsg, http.StatusInternalServerError)
				return
			}
		}

		if !toolCalled {
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Answer: answer,
		Thought: "Upgraded to modern SDK.",
	})
}
