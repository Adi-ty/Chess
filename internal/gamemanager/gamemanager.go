package gamemanager

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GameManager struct {
	games       map[string]*Game
	sessions    map[string]*PlayerSession

	pendingUser string

	mu          sync.RWMutex
}

func NewGameManager() *GameManager {
	return &GameManager{
		games:       make(map[string]*Game),
		sessions:    make(map[string]*PlayerSession),
	}
}

func (gm *GameManager) CanUserConnect(userID string) error {
    gm.mu.RLock()
    defer gm.mu.RUnlock()

    if userID == "" {
        return errors.New("authentication required")
    }

    return nil
}

func (gm *GameManager) AddUser(conn *websocket.Conn, userID string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	session, exists := gm.sessions[userID]
	if !exists {
		session = &PlayerSession{
			UserID: userID,
		}
		gm.sessions[userID] = session
	}

	if session.Conn != nil && session.Conn != conn {
		session.Conn.Close()
	}

	session.Conn = conn
	session.Disconnected = false
	session.LastSeen = time.Now()

	if session.GameID != "" {
		if game, exists := gm.games[session.GameID]; !exists || !game.IsActive() {
			session.GameID = ""
		}
	}

	go gm.AddHandler(session)
}

func (gm *GameManager) RemoveUser(userID string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	session, ok := gm.sessions[userID]
	if !ok {
		return
	}
	session.Conn = nil
	session.Disconnected = true
	session.LastSeen = time.Now()

	if session.GameID != "" {
		game := gm.games[session.GameID]
		if game != nil {
			game.HandleDisconnect(session.UserID, gm)

			game.mu.RLock()
            if game.status == GameStatusAbandoned {
                delete(gm.games, session.GameID)
            }
            game.mu.RUnlock()
		}
	}

	log.Printf("User %s disconnected", userID)
}

func (gm *GameManager) AddHandler(session *PlayerSession) {
	defer func() {
		if session.Conn != nil {
			session.Conn.Close()
		}
		gm.RemoveUser(session.UserID)
	}()

	for {
		if session.Conn == nil {
			return
		}

		_, rawMsg, err := session.Conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}

		var message IncomingMessage
		if err := json.Unmarshal(rawMsg, &message); err != nil {
			session.Conn.WriteJSON(OutgoingError{Type: ERROR, Message: "invalid message format"})
			continue
		}

		gm.handleMessage(session, message)
	}
}

func (gm *GameManager) handleMessage(session *PlayerSession, message IncomingMessage) {
	switch message.Type {
	case INIT_GAME:
		gm.handleInitGame(session)
	case MOVE:
		gm.handleMove(session, message.Move)
	default:
		session.Conn.WriteJSON(OutgoingError{Type: ERROR, Message: "unknown message type"})
	}
}

func (gm *GameManager) handleInitGame(session *PlayerSession) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if existingGame, exists := gm.games[session.GameID]; exists {
		if existingGame.IsActive() {
			session.Conn.WriteJSON(OutgoingError{
				Type:    ERROR,
				Message: "you are already in an active game",
			})
			return
		} else {
			delete(gm.games, session.GameID)
			session.GameID = ""
		}
	}

	if gm.pendingUser == session.UserID {
		session.Conn.WriteJSON(OutgoingError{
			Type:    ERROR,
			Message: "already waiting for opponent",
		})
		return
	}

	if gm.pendingUser != "" {
		if _, exists := gm.sessions[gm.pendingUser]; !exists {
			gm.pendingUser = ""
		}
	}

	currentUserID := session.UserID

	if gm.pendingUser != "" {
		pendingUserID := gm.pendingUser

		// Prevent same user from playing against themselves
		if currentUserID != "" && pendingUserID != "" && currentUserID == pendingUserID {
			session.Conn.WriteJSON(OutgoingError{
				Type:    ERROR,
				Message: "you cannot play against yourself",
			})
			return
		}

		gm.pendingUser = ""

		whiteUserID := pendingUserID
		blackUserID := currentUserID

		game := StartNewGame(whiteUserID, blackUserID)
		session.GameID = game.ID
		gm.sessions[pendingUserID].GameID = game.ID
		gm.games[game.ID] = game
		
		gm.sessions[whiteUserID].Conn.WriteJSON(map[string]string{"type": "game_start", "color": "white", "game_id": game.ID})
		gm.sessions[blackUserID].Conn.WriteJSON(map[string]string{"type": "game_start", "color": "black", "game_id": game.ID})

		log.Printf("Game started: %s (white: %s, black: %s)", game.ID, whiteUserID, blackUserID)
	} else {
		gm.pendingUser = session.UserID
		session.Conn.WriteJSON(map[string]string{
			"type":    "waiting",
			"message": "waiting for opponent",
		})
		log.Printf("Player %s waiting for opponent", currentUserID)
	}
}

func (gm *GameManager) handleMove(session *PlayerSession, move string) {
	gm.mu.RLock()
	game, exists := gm.games[session.GameID]
	gm.mu.RUnlock()

	if !exists || game == nil {
		session.Conn.WriteJSON(OutgoingError{
			Type:    ERROR,
			Message: "you are not in a game",
		})
		return
	}

	if session.GameID == "" {
		session.Conn.WriteJSON(OutgoingError{Type: ERROR, Message: "no active game"})
		return
	}

	if err := game.MakeMove(session, move, gm); err != nil {
		session.Conn.WriteJSON(OutgoingError{
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
	return len(gm.sessions)
}

