package gamemanager

import (
	"time"

	"github.com/gorilla/websocket"
)

type PlayerSession struct {
	UserID       string
	Conn         *websocket.Conn
	GameID       string
	Disconnected bool
	DisconnectedAt time.Time
	LastSeen     time.Time
}