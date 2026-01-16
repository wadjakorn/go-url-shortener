package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	oauthConfig   *oauth2.Config
	jwtSecret     []byte
	frontendURL   string
	allowedEmails []string
	isProduction  bool
}

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		oauthConfig: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
		jwtSecret:     []byte(cfg.JWTSecret),
		frontendURL:   cfg.FrontendURL,
		allowedEmails: cfg.AllowedEmails,
		isProduction:  cfg.AppEnv == "production",
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	state := h.generateStateOauthCookie(w)
	url := h.oauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	oauthState, err := r.Cookie("oauthstate")
	if err != nil {
		log.Printf("Callback error: missing oauthstate cookie: %v", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	if r.FormValue("state") != oauthState.Value {
		log.Printf("Callback error: invalid oauth state. Expected %s, got %s", oauthState.Value, r.FormValue("state"))
		http.Error(w, "invalid oauth google state", http.StatusOK)
		return
	}

	code := r.FormValue("code")
	token, err := h.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Callback error: code exchange failed: %v", err)
		http.Error(w, "code exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		log.Printf("Callback error: failed getting user info: %v", err)
		http.Error(w, "failed getting user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	var googleUser GoogleUser
	if err := json.NewDecoder(response.Body).Decode(&googleUser); err != nil {
		log.Printf("Callback error: failed decoding user info: %v", err)
		http.Error(w, "failed decoding user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Email Allowlist Check
	if len(h.allowedEmails) > 0 {
		isAllowed := false
		for _, email := range h.allowedEmails {
			if email == googleUser.Email {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			log.Printf("Callback error: email %s not in allowlist", googleUser.Email)
			http.Error(w, "Access denied: your email is not in the allowlist", http.StatusForbidden)
			return
		}
	}

	// Create JWT Token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &jwt.RegisteredClaims{
		Subject:   googleUser.Email,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := jwtToken.SignedString(h.jwtSecret)
	if err != nil {
		log.Printf("Callback error: failed signing JWT: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Set Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Expires:  expirationTime,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.isProduction, // Set based on environment
		SameSite: http.SameSiteLaxMode,
	})

	log.Printf("Login successful for user: %s", googleUser.Email)
	// Redirect to frontend/dashboard
	http.Redirect(w, r, h.frontendURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.isProduction,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, h.frontendURL+"/login", http.StatusTemporaryRedirect)
}

func (h *AuthHandler) generateStateOauthCookie(w http.ResponseWriter) string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{
		Name:     "oauthstate",
		Value:    state,
		Expires:  time.Now().Add(20 * time.Minute),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.isProduction, // Set based on environment
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
	return state
}
