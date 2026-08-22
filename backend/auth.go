package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var googleOauthConfig *oauth2.Config

const oauthStateString = "random-string"

func initOAuth() {
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URI"),
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes: []string{"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/calendar.readonly"},
		Endpoint: google.Endpoint,
	}
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	// Generate google login URL and redirect the user there
	url := googleOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Google sends the user back here with a "code". We exchange this code for an access token.
	if r.FormValue("state") != oauthStateString {
		http.Error(w, "Invalid OAuth State", http.StatusBadRequest)
		return
	}

	token, err := googleOauthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		http.Error(w, "Code exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 1. Get user info
	userInfo, err := getUserInfoFromGoogle(token.AccessToken)
	if err != nil {
		http.Error(w, "Failed to fetch user info", http.StatusInternalServerError)
		return
	}

	googleId := userInfo["id"].(string)
	email := userInfo["email"].(string)

	// 2. Save everyuthing to sqlite db
	db := InitDB()
	defer db.Close()

	err = SaveUser(db, googleId, email, token.AccessToken, token.RefreshToken, token.Expiry)
	if err != nil {
		http.Error(w, "Failed to save user to database: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Saved user %s to db\n", email)

	// Test fetching calendar events and tasks
	FetchCalendarEvents(db, token.AccessToken, token.RefreshToken, token.Expiry, googleId)

	// 3. Respond to the user
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful and user saved",
		"email":   email,
	})
}

func getUserInfoFromGoogle(token string) (map[string]interface{}, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}
	return userInfo, nil
}
