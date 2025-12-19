package app

import (
	"log"
	"os"

	"github.com/Adi-ty/chess/internal/api"
	"github.com/Adi-ty/chess/internal/gamemanager"
)

type Application struct {
	Logger *log.Logger
	WebSocketHandler *api.WebSocketHandler
}

func NewApplication() (*Application, error) {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	gm := gamemanager.NewGameManager()

	websocketHandler := api.NewWebSocketHandler(logger, gm)

	app := &Application{
		Logger: logger,
		WebSocketHandler: websocketHandler,
	}

	return app, nil
}