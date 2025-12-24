package gamemanager

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/Adi-ty/chess/internal/store"
	"github.com/gorilla/websocket"
	"github.com/notnil/chess"
	"github.com/redis/go-redis/v9"
)

type GameManager struct {
	games       map[string]*Game
	sessions    map[string]*PlayerSession

	pendingUser string

	gameStore  store.GameStore
	redisClient *redis.Client

	pubsubs map[string]*redis.PubSub

	mu          sync.RWMutex
}

func NewGameManager(gameStore store.GameStore, redisClient *redis.Client) *GameManager {
	return &GameManager{
		games:       make(map[string]*Game),
		sessions:    make(map[string]*PlayerSession),
		gameStore:   gameStore,
		redisClient: redisClient,
		pubsubs:     make(map[string]*redis.PubSub),
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
		// if game, exists := gm.games[session.GameID]; exists && game.IsActive() {
        //     // Game is in memory, no need to fetch/replay
		// 	game.mu.RLock()
        //     boardState := game.board.Position().Board().String()
        //     game.mu.RUnlock()
        //     session.Conn.WriteJSON(map[string]interface{}{
        //         "type": "board_state",
        //         "board": boardState,
        //         "game_id": game.ID,
        //     })
        //     go gm.AddHandler(session)
        //     return
        // }

		dbGame, err := gm.gameStore.GetGameByUserID(context.Background(), userID)
		if err != nil {
			log.Printf("Failed to fetch game from store: %v", err)
		} else if dbGame != nil {
			game := &Game{
                ID:          dbGame.ID,
                WhiteUserID: dbGame.WhiteUserID,
                BlackUserID: dbGame.BlackUserID,
                board:       chess.NewGame(),
                status:      GameStatusInProgress,
                startTime:   time.Now(),
                moveNumber:  0,
                disconnected: make(map[string]time.Time),
            }
			gm.games[dbGame.ID] = game
			gm.sessions[game.WhiteUserID].GameID = game.ID
            gm.sessions[game.BlackUserID].GameID = game.ID

			// Replay moves
			moves, err := gm.gameStore.GetMovesByGameID(context.Background(), dbGame.ID)
			if err != nil {
                log.Printf("Error fetching moves for game %s: %v", dbGame.ID, err)
            } else if len(moves) > 0 {
				if whiteSess, exists := gm.sessions[game.WhiteUserID]; exists && whiteSess.Conn != nil {
                    whiteSess.Conn.WriteJSON(map[string]interface{}{"type": "board_replay", "moves": moves})
                }
                if blackSess, exists := gm.sessions[game.BlackUserID]; exists && blackSess.Conn != nil {
                    blackSess.Conn.WriteJSON(map[string]interface{}{"type": "board_replay", "moves": moves})
                }

			
				if _, exists := gm.pubsubs[dbGame.ID]; !exists {
					pubsub := gm.redisClient.Subscribe(context.Background(), "game:"+dbGame.ID)
					gm.pubsubs[dbGame.ID] = pubsub
					go gm.listenForMoves(dbGame.ID)
					log.Printf("Subscribed to game channel: %s", dbGame.ID)
				}

                for _, move := range moves {
					mv, err := chess.UCINotation{}.Decode(game.board.Position(), move.Move)
					if err != nil {
						session.Conn.WriteJSON(OutgoingError{Type: ERROR, Message: "failed to restore game"})
						log.Printf("Error decoding move %s: %v", move.Move, err)
						return
					}
					
                    if err := game.board.Move(mv); err != nil {
                        log.Printf("Error replaying move %s: %v", move.Move, err)
						session.Conn.WriteJSON(OutgoingError{Type: ERROR, Message: "failed to restore game"})
						return
                    }
                    game.moveNumber = move.MoveNumber
                }
            }
		}

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
                if pubsub, exists := gm.pubsubs[session.GameID]; exists {
                    pubsub.Close()
                    delete(gm.pubsubs, session.GameID)
                }
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
		gm.games[game.ID] = game
		gm.sessions[whiteUserID].GameID = game.ID
		gm.sessions[blackUserID].GameID = game.ID

		pubsub := gm.redisClient.Subscribe(context.Background(), "game:"+game.ID)
		gm.pubsubs[game.ID] = pubsub

		go gm.listenForMoves(game.ID)

		_, err := gm.gameStore.CreateGame(context.Background(), &store.Game{
			ID:          game.ID,
			WhiteUserID: whiteUserID,
			BlackUserID: blackUserID,
			Status:      string(GameStatusInProgress),
			StartedAt:   game.startTime.Format(time.RFC3339),
		})
		if err != nil {
			log.Printf("Failed to create game in store: %v", err)
		}
		
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

func (gm *GameManager) listenForMoves(gameID string) {
    pubsub := gm.pubsubs[gameID]
    defer pubsub.Close()

    ch := pubsub.Channel()
    for msg := range ch {
        var moveMsg OutgoingMove
        if err := json.Unmarshal([]byte(msg.Payload), &moveMsg); err != nil {
            log.Printf("Error unmarshaling pubsub message: %v", err)
            continue
        }

        game := gm.games[gameID]
        if game != nil {
            game.mu.RLock()
            game.safeSend(gm.sessions[game.WhiteUserID].Conn, moveMsg)
            game.safeSend(gm.sessions[game.BlackUserID].Conn, moveMsg)
            game.mu.RUnlock()
        }
    }
}