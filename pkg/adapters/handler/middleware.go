package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/config"
)

type Middleware struct {
	jwtSecret []byte
}

func NewMiddleware(cfg *config.Config) *Middleware {
	return &Middleware{
		jwtSecret: []byte(cfg.JWTSecret),
	}
}

// AuthMiddleware verifies the JWT token from the cookie
func (m *Middleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for auth_token cookie
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			if isAPIRequest(r) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/auth/google/login", http.StatusTemporaryRedirect)
			}
			return
		}

		tokenString := cookie.Value
		claims := &jwt.RegisteredClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return m.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			if isAPIRequest(r) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/auth/google/login", http.StatusTemporaryRedirect)
			}
			return
		}

		// Token is valid, proceed
		ctx := context.WithValue(r.Context(), "user_email", claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isAPIRequest(r *http.Request) bool {
	// Simple heuristic: check if path starts with /api/
	// or Content-Type is application/json
	return strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/v1/dashboard" // Dashboard might be viewed in browser
}
