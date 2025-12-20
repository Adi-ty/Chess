package routes

import (
	"net/http"

	"github.com/Adi-ty/chess/internal/app"
)

func SetUpRoutes(app *app.Application) *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/ws", app.WebSocketHandler.WsHandler)
	router.HandleFunc("GET /auth/google", app.AuthHandler.HandleGoogleLogin)
	router.HandleFunc("GET /auth/google/callback", app.AuthHandler.HandleGoogleCallback)
	router.HandleFunc("POST /auth/logout", app.AuthHandler.HandleLogout)

	router.Handle("GET /auth/me", app.JWTService.Middleware(
		http.HandlerFunc(app.AuthHandler.HandleMe),
	))

	return router
}