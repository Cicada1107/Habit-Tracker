package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var googleOauthConfig *oauth2.Config

// ToDo: put these in .env file and load them using godotenv
const oauthStateString = "random-string"

var jwtSecretKey = []byte("secret-habit-coach-jwt-key")

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
	// Generate google login URL and redirect the user there. 
	// IMPORTANT: We MUST request offline access to get a refresh token!
	url := googleOauthConfig.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
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

	// 2. Save everything to sqlite db
	db := InitDB()
	defer db.Close()

	err = SaveUser(db, googleId, email, token.AccessToken, token.RefreshToken, token.Expiry)
	if err != nil {
		http.Error(w, "Failed to save user to database: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// look up the iternal user id for the given google id
	var internalUserID string
	db.QueryRow("SELECT id FROM users WHERE google_id =?", googleId).Scan(&internalUserID)

	// 🔥 Trigger a massive 7-day sync immediately on login!
	// This guarantees any tasks missed by broken webhooks are instantly caught up.
	EnqueueSyncJob(googleId)

	// 3. Generate JWT token for the user
	tokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": internalUserID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
	})

	// sign it with secret key
	signedToken, err := tokenJWT.SignedString(jwtSecretKey)
	if err != nil {
		http.Error(w, "Failed to sign JWT token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// set the token as the secure HTTP-Only cookie in the user's browser
	http.SetCookie(w, &http.Cookie{
		Name:     "habit_session_token",
		Value:    signedToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	// redirect the user to the react frontend app
	http.Redirect(w, r, "http://localhost:5173", http.StatusTemporaryRedirect)
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
