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
	"github.com/Adi-ty/chess/internal/worker"
	"github.com/Adi-ty/chess/migrations"
	"github.com/redis/go-redis/v9"
)

type Application struct {
	Logger *log.Logger
	Config *config.Config
	AuthHandler *api.AuthHandler
	WebSocketHandler *api.WebSocketHandler
	JWTService       *auth.JWTService
	DB *sql.DB
	redisClient *redis.Client
	worker *worker.Worker
}

func NewApplication() (*Application, error) {
	pgDB, err := store.Open()
	if err != nil {
		return nil, err
	}

	redisDB, err := store.OpenRedis()
	if err != nil {
		return nil, err
	}
	
	err = store.MigrateFS(pgDB, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	cfg := config.LoadConfig()

	// Stores
	userStore := store.NewPostgresUserStore(pgDB)
	gameStore := store.NewPostgresGameStore(pgDB)

	// Services
	gm := gamemanager.NewGameManager(gameStore, redisDB)

	jwtService := auth.NewJWTService(cfg.JWTSecret)
	googleOauth := auth.NewGoogleOAuth(&auth.GoogleConfig{
		ClientID: cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURI: cfg.GoogleRedirectURI,
	})

	// Handlers
	authHandler := api.NewAuthHandler(logger, googleOauth, jwtService, userStore)
	websocketHandler := api.NewWebSocketHandler(logger, gm, jwtService)

	// Start worker go-routine
	wk := worker.NewWorker(redisDB, gameStore)
	go wk.Start()

	app := &Application{
		Logger: logger,
		Config: cfg,
		AuthHandler: authHandler,
		WebSocketHandler: websocketHandler,
		JWTService: jwtService,
		DB: pgDB,
		redisClient: redisDB,
		worker: wk,
	}

	return app, nil
}