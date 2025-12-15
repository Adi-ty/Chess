package gamemanager

import (
	"time"

	"github.com/gorilla/websocket"
)

type Game struct {
	player1 *websocket.Conn
	player2 *websocket.Conn
	board   string
	moves  []string
	startTime time.Time
}

func StartNewGame(player1, player2 *websocket.Conn) *Game {
	return &Game{
		player1: player1,
		player2: player2,
		board:   "",
		moves:   []string{},
		startTime: time.Now(),
	}
}