package routes

import (
	"net/http"

	"github.com/Adi-ty/chess/internal/app"
)

func SetUpRoutes(app *app.Application) *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/ws", app.WebSocketHandler.WsHandler)

	return router
}