package main

import (
	"fmt"
	"net/http"

	"github.com/Adi-ty/chess/internal/gamemanager"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var gm = gamemanager.NewGameManager()

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrade error:", err)
		return
	}

	go handleConnection(conn)
}

func handleConnection(conn *websocket.Conn) {
	defer conn.Close()

	gm.AddUser(conn)
	defer gm.RemoveUser(conn)

	gm.AddHandler(conn)
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	fmt.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
