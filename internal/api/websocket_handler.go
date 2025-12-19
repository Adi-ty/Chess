package api

import (
	"log"
	"net/http"

	"github.com/Adi-ty/chess/internal/auth"
	"github.com/Adi-ty/chess/internal/gamemanager"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	logger *log.Logger
	gamemanager *gamemanager.GameManager
	jwtService  *auth.JWTService
}

func NewWebSocketHandler(logger *log.Logger, gm *gamemanager.GameManager, jwtService *auth.JWTService) *WebSocketHandler {
	return &WebSocketHandler{
		logger: logger,
		gamemanager: gm,
		jwtService: jwtService,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *WebSocketHandler) WsHandler(w http.ResponseWriter, r *http.Request) {
    tokenString := r.URL.Query().Get("token")

	if tokenString == "" {
        if cookie, err := r.Cookie("auth_token"); err == nil {
            tokenString = cookie.Value
        }
    }

	
	claims, err := h.jwtService.ValidateToken(tokenString)
	if err != nil {
		h.logger.Printf("Invalid token: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        h.logger.Printf("Upgrade error: %v", err)
        return
    }

    go func(conn *websocket.Conn, userID string) {
        defer conn.Close()
        defer h.gamemanager.RemoveUser(conn)
        
        h.gamemanager.AddUser(conn, userID)
    }(conn, userID)
}