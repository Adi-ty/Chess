package gamemanager

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/notnil/chess"
)

type GameStatus string

const (
	GameStatusInProgress GameStatus = "in_progress"
	GameStatusCompleted  GameStatus = "completed"
	GameStatusAbandoned  GameStatus = "abandoned"
)

var (
	ErrGameEnded   = errors.New("game has already ended")
	ErrNotYourTurn = errors.New("not your turn")
	ErrInvalidMove = errors.New("invalid move format")
	ErrNotInGame   = errors.New("you are not in this game")
	ErrEmptyMove   = errors.New("move cannot be empty")
)

type Game struct {
	ID        string

	WhiteUserID string
	BlackUserID string

	board     *chess.Game
	status    GameStatus

	startTime time.Time
	endTime   time.Time

	disconnected map[string]time.Time 

	mu        sync.RWMutex
}

func StartNewGame(whiteUserID, blackUserID string) *Game {
	return &Game{
		ID:        uuid.New().String(),
		WhiteUserID: whiteUserID,
		BlackUserID: blackUserID,
		board:     chess.NewGame(),
		status:    GameStatusInProgress,
		startTime: time.Now(),
		disconnected: make(map[string]time.Time),
	}
}

func (g *Game) MakeMove(session *PlayerSession, move string, gm *GameManager) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status != GameStatusInProgress {
		return ErrGameEnded
	}

	if move == "" {
		return ErrEmptyMove
	}

	if session.UserID != g.WhiteUserID && session.UserID != g.BlackUserID {
		return ErrNotInGame
	}

	turn := g.board.Position().Turn()
	if (turn == chess.White && session.UserID != g.WhiteUserID) || (turn == chess.Black && session.UserID != g.BlackUserID) {
		return ErrNotYourTurn
	}

	mv, err := chess.UCINotation{}.Decode(g.board.Position(), move)
	if err != nil {
		return ErrInvalidMove
	}

	if err := g.board.Move(mv); err != nil {
		return ErrInvalidMove
	}

	outcome := g.board.Outcome()
	if outcome != chess.NoOutcome {
		g.status = GameStatusCompleted
		g.endTime = time.Now()

		err := gm.gameStore.UpdateGameStatus(context.Background(), g.ID, string(GameStatusCompleted), outcome.String(), g.board.Method().String(), g.endTime.Format(time.RFC3339))
		if err != nil {
			log.Printf("Failed to update game status in store: %v", err)
		}

		gameOverMsg := OutgoingGameOver{
			Type:    GAME_OVER,
			Outcome: outcome.String(),
			Method:  g.board.Method().String(),
		}

		g.safeSend(gm.sessions[g.WhiteUserID].Conn, gameOverMsg)
		g.safeSend(gm.sessions[g.BlackUserID].Conn, gameOverMsg)
		return nil
	}

	moveMsg := OutgoingMove{Type: MOVE, Move: move}
	g.safeSend(gm.sessions[g.WhiteUserID].Conn, moveMsg)
	g.safeSend(gm.sessions[g.BlackUserID].Conn, moveMsg)

	return nil
}

func (g *Game) HandleDisconnect(userID string, gm *GameManager) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status != GameStatusInProgress {
		return
	}
	g.disconnected[userID] = time.Now()

	session, ok := gm.sessions[userID]
	if ok {
		session.DisconnectedAt = time.Now()
	}

	go func() {
		time.Sleep(15 * time.Second)
		g.mu.Lock()
		defer g.mu.Unlock()

		session, ok := gm.sessions[userID]
		if !ok || !session.Disconnected {
			return
		}

		if g.status == GameStatusInProgress {
			g.status = GameStatusAbandoned
			g.endTime = time.Now()

			err := gm.gameStore.UpdateGameStatus(context.Background(), g.ID, string(g.status), string(GameStatusAbandoned), "disconnect", g.endTime.Format(time.RFC3339))
			if err != nil {
				log.Printf("Failed to update game status in store: %v", err)
			}

			abandonMsg := OutgoingGameOver{
				Type:    GAME_OVER,
				Outcome: "abandoned",
				Method:  "disconnect",
			}
			
			g.safeSend(gm.sessions[g.WhiteUserID].Conn, abandonMsg)
			g.safeSend(gm.sessions[g.BlackUserID].Conn, abandonMsg)

			if whiteSess, exists := gm.sessions[g.WhiteUserID]; exists {
                whiteSess.GameID = ""
            }
            if blackSess, exists := gm.sessions[g.BlackUserID]; exists {
                blackSess.GameID = ""
            }
		}
	}()
}


func (g *Game) IsActive() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.status == GameStatusInProgress
}

func (g *Game) safeSend(conn *websocket.Conn, msg interface{}) {
	if conn == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			// Connection was closed, ignore
		}
	}()
	conn.WriteJSON(msg)
}