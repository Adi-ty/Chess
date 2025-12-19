package api

import (
	"log"
	"net/http"

	"github.com/Adi-ty/chess/internal/gamemanager"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	logger *log.Logger
	gamemanager *gamemanager.GameManager
}

func NewWebSocketHandler(logger *log.Logger, gm *gamemanager.GameManager) *WebSocketHandler {
	return &WebSocketHandler{
		logger: logger,
		gamemanager: gm,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *WebSocketHandler) WsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        h.logger.Printf("Upgrade error: %v", err)
        return
    }

    go func(conn *websocket.Conn) {
        defer conn.Close()
        defer h.gamemanager.RemoveUser(conn)
        
        h.gamemanager.AddUser(conn)
    }(conn)
}