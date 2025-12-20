package gamemanager

import (
	"encoding/json"
	"errors"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type GameManager struct {
	games       map[string]*Game
	playerGames map[*websocket.Conn]*Game
	pendingUser *websocket.Conn
	users       map[*websocket.Conn]bool

	connToUser  map[*websocket.Conn]string
    userToConn  map[string]*websocket.Conn

	mu          sync.RWMutex
}

func NewGameManager() *GameManager {
	return &GameManager{
		games:       make(map[string]*Game),
		playerGames: make(map[*websocket.Conn]*Game),
		pendingUser: nil,
		users:       make(map[*websocket.Conn]bool),
		connToUser:  make(map[*websocket.Conn]string),
        userToConn:  make(map[string]*websocket.Conn),
	}
}

func (gm *GameManager) CanUserConnect(userID string) error {
    gm.mu.RLock()
    defer gm.mu.RUnlock()

    if userID == "" {
        return errors.New("authentication required")
    }

    if existingConn, exists := gm.userToConn[userID]; exists {
        if game, inGame := gm.playerGames[existingConn]; inGame && game.IsActive() {
            return errors.New("user is already in an active game")
        }
        return errors.New("user already has an active connection")
    }

    return nil
}

func (gm *GameManager) AddUser(conn *websocket.Conn, userID string) {
	gm.mu.Lock()
	gm.users[conn] = true

	if userID != "" {
        if oldConn, exists := gm.userToConn[userID]; exists && oldConn != conn {
			if game, inGame := gm.playerGames[oldConn]; !inGame || !game.IsActive() {
				if gm.pendingUser == oldConn {
					gm.pendingUser = nil
					log.Printf("Cleared pending user due to reconnection: %s", userID)
				}
			}
			

            delete(gm.connToUser, oldConn)
            delete(gm.users, oldConn)
			oldConn.Close()
        }
        
        gm.connToUser[conn] = userID
        gm.userToConn[userID] = conn
    }
	gm.mu.Unlock()

	gm.AddHandler(conn)
}

func (gm *GameManager) RemoveUser(conn *websocket.Conn) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if userID, exists := gm.connToUser[conn]; exists {
       	if gm.userToConn[userID] == conn {
            delete(gm.userToConn, userID)
        }
        delete(gm.connToUser, conn)
    }

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

	currentUserID := gm.connToUser[conn]

	if gm.pendingUser != nil {
		pendingUserID := gm.connToUser[gm.pendingUser]

		// Prevent same user from playing against themselves
		if currentUserID != "" && pendingUserID != "" && currentUserID == pendingUserID {
			conn.WriteJSON(OutgoingError{
				Type:    ERROR,
				Message: "you cannot play against yourself",
			})
			return
		}

		opponent := gm.pendingUser
		gm.pendingUser = nil

		whiteUserID := gm.connToUser[opponent]
		blackUserID := gm.connToUser[conn]

		game := StartNewGame(opponent, conn, whiteUserID, blackUserID)
		gm.games[game.ID] = game
		gm.playerGames[opponent] = game
		gm.playerGames[conn] = game

		log.Printf("Game started: %s (white: %s, black: %s)", game.ID, whiteUserID, blackUserID)
	} else {
		gm.pendingUser = conn
		conn.WriteJSON(map[string]string{
			"type":    "waiting",
			"message": "waiting for opponent",
		})
		log.Printf("Player %s waiting for opponent", currentUserID)
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

