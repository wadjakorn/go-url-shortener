package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/config"
)

func TestAuthMiddleware(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "testservlet",
	}
	mw := NewMiddleware(cfg)

	tests := []struct {
		name           string
		path           string
		cookieName     string
		cookieValue    string
		expectedStatus int
	}{
		{
			name:           "No Cookie - API",
			path:           "/api/v1/links",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "No Cookie - Browser",
			path:           "/dashboard",
			expectedStatus: http.StatusTemporaryRedirect,
		},
		{
			name:           "Invalid Cookie - API",
			path:           "/api/v1/links",
			cookieName:     "auth_token",
			cookieValue:    "invalid",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Valid Cookie - API",
			path:           "/api/v1/links",
			cookieName:     "auth_token",
			cookieValue:    generateTestToken(t, cfg.JWTSecret),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.cookieName != "" {
				req.AddCookie(&http.Cookie{Name: tt.cookieName, Value: tt.cookieValue})
			}

			rr := httptest.NewRecorder()
			handler := mw.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}
		})
	}
}

func generateTestToken(t *testing.T, secret string) string {
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &jwt.RegisteredClaims{
		Subject:   "test@example.com",
		ExpiresAt: jwt.NewNumericDate(expirationTime),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}
	return tokenString
}
