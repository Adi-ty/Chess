package app

import (
	"database/sql"
	"log"
	"os"

	"github.com/Adi-ty/chess/internal/api"
	"github.com/Adi-ty/chess/internal/auth"
	"github.com/Adi-ty/chess/internal/config"
	"github.com/Adi-ty/chess/internal/gamemanager"
	"github.com/Adi-ty/chess/internal/store"
	"github.com/Adi-ty/chess/migrations"
)

type Application struct {
	Logger *log.Logger
	AuthHandler *api.AuthHandler
	WebSocketHandler *api.WebSocketHandler
	DB *sql.DB
}

func NewApplication() (*Application, error) {
	pgDB, err := store.Open()
	if err != nil {
		return nil, err
	}

	
	err = store.MigrateFS(pgDB, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	cfg := config.LoadConfig()

	gm := gamemanager.NewGameManager()

	jwtService := auth.NewJWTService(cfg.JWTSecret)
	googleOauth := auth.NewGoogleOAuth(&auth.GoogleConfig{
		ClientID: cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURI: cfg.GoogleRedirectURI,
	})

	// Handlers
	authHandler := api.NewAuthHandler(logger, googleOauth, jwtService)
	websocketHandler := api.NewWebSocketHandler(logger, gm)

	app := &Application{
		Logger: logger,
		AuthHandler: authHandler,
		WebSocketHandler: websocketHandler,
		DB: pgDB,
	}

	return app, nil
}