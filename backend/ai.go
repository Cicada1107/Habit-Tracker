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
func handlehabitCoachChat(w http.ResponseWriter, r *http.Request) {
	// 1. Get the user's question from frontend
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 2. Grab user id from the request context
	userID := r.Context().Value("user_id").(string)

	// 3. Fetch their habit data to send to the AI as context
	db := InitDB()
	defer db.Close()

	monthAgo := time.Now().AddDate(0, -1, 0)
	// toDo: Implement fetchUserHabits function to get user's habit data of last x days from the database in database.go
	habits, err := fetchUserHabits(db, userID, monthAgo)
	if err != nil {
		http.Error(w, "Failed to fetch user habits", http.StatusInternalServerError)
		return
	}
	dataContext := "Here is the user's habit data for the last 6 months:\n"
	for habit, mins := range habits {
		dataContext += fmt.Sprintf("- Habit: %s, Total Minutes: %d\n", habit, mins)
	}

	// 4. Initalize the AI agent with the user's habit data and the question
	ctx := r.Context()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		http.Error(w, "Failed to initialize AI client", http.StatusInternalServerError)
		return
	}

	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text("You are a personal computerized Habit Coach AI." + "You analyze the user's provided calendar data and answer their queries." + "If they ask about probabilities, calculate the likelihood of the query based on their event data and past history" + "Use as accurate and sophisticated statistical methods as possible." + "Keep your answers concise, actionable and formatted in markdown." + "If you don't have enough information to answer, say 'I don't know' and ask for more information." + "Do not make up answers or provide false information." + "Do not provide generic advice. Only provide advice based on the user's data." + "Do not provide any medical advice. If the user asks for medical advice, say 'I am not a medical professional. Please consult a doctor for medical advice.'"),
		},
	}

	// 5. Create a prompt with the user's question and their habit data and send it
	prompt := fmt.Sprintf("%s\n\nUser Question: %s", dataContext, req.Question)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		http.Error(w, "Failed to generate AI response", http.StatusInternalServerError)
		return
	}

	// 6. Extract and return the AI's answer to the frontend
	var answer string
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		if text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
			answer = string(text)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChartResponse{Answer: answer})
}
