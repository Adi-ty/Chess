package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Adi-ty/chess/internal/auth"
	"github.com/Adi-ty/chess/internal/store"
)

type AuthHandler struct {
	logger *log.Logger
	googleOAuth *auth.GoogleOAuth
	jwtService *auth.JWTService
	userStore store.UserStore
}

func NewAuthHandler(
	logger *log.Logger,
	googleOAuth *auth.GoogleOAuth,
	jwtService *auth.JWTService,
	userStore store.UserStore,
) *AuthHandler {
	return &AuthHandler{
		logger: logger,
		googleOAuth: googleOAuth,
		jwtService: jwtService,
		userStore: userStore,
	}
}

func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
    state := generateState()

    http.SetCookie(w, &http.Cookie{
        Name:     "oauth_state",
        Value:    state,
        Path:     "/",
        MaxAge:   300,
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
    })

    authURL := h.googleOAuth.GetAuthURL(state)
    http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "Missing state", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name: "oauth_state",
		Value: "",
		Path: "/",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	tokenResp, err := h.googleOAuth.ExchangeCode(r.Context(), code)
	if err != nil {
		h.logger.Printf("Failed to excahnge code: %v", err)
		http.Error(w, "authenticaton failed", http.StatusInternalServerError)
		return
	}

	userInfo, err := h.googleOAuth.GetUserInfo(r.Context(), tokenResp.AccessToken)
	if err != nil {
		h.logger.Printf("Failed to get user info: %v", err)
		http.Error(w, "failed to get user info", http.StatusInternalServerError)
		return
	}

	user, err := h.userStore.CreateOrUpdate(r.Context(), &store.User{
		Email: userInfo.Email,
		DisplayName: userInfo.Name,
		AvatarURL: userInfo.Picture,
		Provider: "google",
		ProviderID: userInfo.ID,
	})
	if err != nil {
		h.logger.Printf("Failed to create/update user: %v", err)
		http.Error(w, "failed to save user", http.StatusInternalServerError)
		return
	}

	token, err := h.jwtService.GenerateToken(user.ID, user.Email, 24 * time.Hour)
	if err != nil {
		h.logger.Printf("Failed to generate token: %v", err)
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    token,
        Path:     "/",
        MaxAge:   86400,
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
    })

	http.Redirect(w, r, "http://localhost:3000/auth/callback?token="+token, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUserFromContext(r.Context())
	if userCtx == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	user, err := h.userStore.GetUserByID(r.Context(), userCtx.UserID)
	if err != nil {
		h.logger.Printf("Failed to get user: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "user not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
    http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    "",
        Path:     "/",
        MaxAge:   -1,
        HttpOnly: true,
    })

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"message": "logged out"})
}

func generateState() string {
    b := make([]byte, 16)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}