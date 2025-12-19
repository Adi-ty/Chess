package api

import (
	"log"

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
