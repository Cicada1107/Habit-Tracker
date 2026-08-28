package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

	// Fetch the user's defined habits to give the AI context
	rows, _ := db.Query("SELECT name, description FROM habits WHERE user_id = ?", userID)
	var habitsList string
	for rows.Next() {
		var name string
		var desc sql.NullString
		rows.Scan(&name, &desc)
		habitsList += fmt.Sprintf("- %s (Description: %s)\n", name, desc.String)
	}
	rows.Close()

	if habitsList == "" {
		habitsList = "No habits defined yet."
	}

	currentTime := time.Now().Format(time.RFC3339)
	systemPrompt := fmt.Sprintf(`You are the 'Habit Coach' AI. The current time is %s. You have access to the following tools:
		1. get_events_in_range: Fetches a user's events from the database for a given time range.
		2. calculate_habit_probability: Calculates the probability of a user adhering to a habit based on past data.

		IMPORTANT: The user tracks specific explicit habits. When using tools, you MUST use the exact 'Habit Name' listed below for the habit_name argument.
		Do NOT use the user's raw query terms (like 'system design') for the habit_name argument; map their query to one of the tracking goals below.

		User's Tracked Habits:
		%s

		Use these tools to provide accurate and helpful responses to the user's questions about their habits and events.
		ALWAYS check stats before answering. Be unbiased, concise, professional and to the point.
		CRITICAL INSTRUCTION: You MUST wrap all of your internal reasoning and step-by-step thinking inside <thought>...</thought> XML tags BEFORE outputting your final answer to the user.`, currentTime, habitsList)

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

	var rawAnswer string
	var actionsLog string

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
				
				actionsLog += fmt.Sprintf("Analysing data using tool: %s...\n", funcCall.Name)

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
				rawAnswer += part.Text
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

	// Extract <thought> tags
	finalAnswer := rawAnswer
	extractedThoughts := actionsLog

	startIdx := strings.Index(rawAnswer, "<thought>")
	endIdx := strings.Index(rawAnswer, "</thought>")
	
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		extractedThoughts += strings.TrimSpace(rawAnswer[startIdx+9 : endIdx])
		finalAnswer = strings.TrimSpace(rawAnswer[:startIdx] + rawAnswer[endIdx+10:])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Answer: finalAnswer,
		Thought: strings.TrimSpace(extractedThoughts),
	})
}
