package gamemanager

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type GameManager struct {
	games       map[string]*Game
	playerGames map[*websocket.Conn]*Game
	pendingUser *websocket.Conn
	users       map[*websocket.Conn]bool
	mu          sync.RWMutex
}

func NewGameManager() *GameManager {
	return &GameManager{
		games:       make(map[string]*Game),
		playerGames: make(map[*websocket.Conn]*Game),
		pendingUser: nil,
		users:       make(map[*websocket.Conn]bool),
	}
}

func (gm *GameManager) AddUser(conn *websocket.Conn) {
	gm.mu.Lock()
	gm.users[conn] = true
	gm.mu.Unlock()

	gm.AddHandler(conn)
}

func (gm *GameManager) RemoveUser(conn *websocket.Conn) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	delete(gm.users, conn)

	if gm.pendingUser == conn {
		gm.pendingUser = nil
		log.Printf("Pending user disconnected")
	}

	if game, exists := gm.playerGames[conn]; exists {
		game.HandleDisconnect(conn)

		delete(gm.playerGames, conn)

		if conn == game.white && game.black != nil {
			delete(gm.playerGames, game.black)
		} else if conn == game.black && game.white != nil {
			delete(gm.playerGames, game.white)
		}

		if !game.IsActive() {
			delete(gm.games, game.ID)
		}
	}
}

func (gm *GameManager) AddHandler(conn *websocket.Conn) {
	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}

		var message IncomingMessage
		if err := json.Unmarshal(rawMsg, &message); err != nil {
			conn.WriteJSON(OutgoingError{Type: ERROR, Message: "invalid message format"})
			continue
		}

		gm.handleMessage(conn, message)
	}
}

func (gm *GameManager) handleMessage(conn *websocket.Conn, message IncomingMessage) {
	switch message.Type {
	case INIT_GAME:
		gm.handleInitGame(conn)
	case MOVE:
		gm.handleMove(conn, message.Move)
	default:
		conn.WriteJSON(OutgoingError{Type: ERROR, Message: "unknown message type"})
	}
}

func (gm *GameManager) handleInitGame(conn *websocket.Conn) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if existingGame, exists := gm.playerGames[conn]; exists {
		if existingGame.IsActive() {
			conn.WriteJSON(OutgoingError{
				Type:    ERROR,
				Message: "you are already in an active game",
			})
			return
		}

		delete(gm.playerGames, conn)
	}

	if gm.pendingUser == conn {
		conn.WriteJSON(OutgoingError{
			Type:    ERROR,
			Message: "already waiting for opponent",
		})
		return
	}

	if gm.pendingUser != nil {
		if _, exists := gm.users[gm.pendingUser]; !exists {
			gm.pendingUser = nil
		}
	}

	if gm.pendingUser != nil {
		opponent := gm.pendingUser
		gm.pendingUser = nil

		game := StartNewGame(opponent, conn)
		gm.games[game.ID] = game
		gm.playerGames[opponent] = game
		gm.playerGames[conn] = game

		log.Printf("Game started: %s", game.ID)
	} else {
		gm.pendingUser = conn
		conn.WriteJSON(map[string]string{
			"type":    "waiting",
			"message": "waiting for opponent",
		})
		log.Printf("Player waiting for opponent")
	}
}

func (gm *GameManager) handleMove(conn *websocket.Conn, move string) {
	gm.mu.RLock()
	game, exists := gm.playerGames[conn]
	gm.mu.RUnlock()

	if !exists || game == nil {
		conn.WriteJSON(OutgoingError{
			Type:    ERROR,
			Message: "you are not in a game",
		})
		return
	}

	if err := game.MakeMove(conn, move); err != nil {
		conn.WriteJSON(OutgoingError{
			Type:    ERROR,
			Message: err.Error(),
		})
	}
}

func (gm *GameManager) GetActiveGamesCount() int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	count := 0
	for _, game := range gm.games {
		if game.IsActive() {
			count++
		}
	}
	return count
}

func (gm *GameManager) GetConnectedUsersCount() int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return len(gm.users)
}

