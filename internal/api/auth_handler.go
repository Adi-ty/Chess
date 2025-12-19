package api

import (
	"log"

	"github.com/Adi-ty/chess/internal/auth"
)

type AuthHandler struct {
	logger *log.Logger
	googleOAuth *auth.GoogleOAuth
	jwtService *auth.JWTService
}

func NewAuthHandler(
	logger *log.Logger,
	googleOAuth *auth.GoogleOAuth,
	jwtService *auth.JWTService,
) *AuthHandler {
	return &AuthHandler{
		logger: logger,
		googleOAuth: googleOAuth,
		jwtService: jwtService,
	}
}