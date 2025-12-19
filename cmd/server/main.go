package main

import (
	"fmt"
	"net/http"

	"github.com/Adi-ty/chess/internal/app"
	"github.com/Adi-ty/chess/internal/routes"
)

func main() {
	app, err := app.NewApplication()
	if err != nil {
		fmt.Println("Error initializing application:", err)
		return
	}

	mux := routes.SetUpRoutes(app)

	server := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}
	fmt.Println("Server started on :8080")
	err = server.ListenAndServe()
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
