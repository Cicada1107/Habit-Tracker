package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware checks for a valid jwt token in the auth header before allowing access to a protected endpoint
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Grab the cookie
		cookie, err := r.Cookie("habit_session_token")
		if err != nil {
			http.Error(w, "Unauthorized: No session token or could not find session token", http.StatusUnauthorized)
			return
		}

		// 2. Validate the JWT token
		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecretKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized: Invalid session token", http.StatusUnauthorized)
			return
		}

		// 3. Extract the user_id and pass it down to the next handler using context
		claims := token.Claims.(jwt.MapClaims)
		userID := claims["user_id"].(string)
		ctx := context.WithValue(r.Context(), "user_id", userID)

		// 4. Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
