package main

import (
	"fmt"
	"net/http"

	"github.com/Adi-ty/chess/internal/app"
	"github.com/Adi-ty/chess/internal/auth"
	"github.com/Adi-ty/chess/internal/routes"
)

func main() {
	app, err := app.NewApplication()
	if err != nil {
		fmt.Println("Error initializing application:", err)
		return
	}
	app.Logger.Println("Server Started")

	mux := routes.SetUpRoutes(app)
	handler := auth.CORSMiddleware(mux)

	server := &http.Server{
		Addr: ":8080",
		Handler: handler,
	}
	err = server.ListenAndServe()
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
