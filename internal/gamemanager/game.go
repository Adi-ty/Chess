package gamemanager

import (
	"errors"
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
	white     *websocket.Conn
	black     *websocket.Conn
	whiteUserID string
	blackUserID string
	board     *chess.Game
	status    GameStatus
	startTime time.Time
	endTime   time.Time
	mu        sync.RWMutex
}

func StartNewGame(player1, player2 *websocket.Conn, whiteUserID, blackUserID string) *Game {
	game := &Game{
		ID:        uuid.New().String(),
		white:     player1,
		black:     player2,
		whiteUserID: whiteUserID,
		blackUserID: blackUserID,
		board:     chess.NewGame(),
		status:    GameStatusInProgress,
		startTime: time.Now(),
	}

	player1.WriteJSON(map[string]string{"type": "game_start", "color": "white", "game_id": game.ID})
	player2.WriteJSON(map[string]string{"type": "game_start", "color": "black", "game_id": game.ID})

	return game
}

func (g *Game) MakeMove(player *websocket.Conn, move string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status != GameStatusInProgress {
		return ErrGameEnded
	}

	if move == "" {
		return ErrEmptyMove
	}

	if player != g.white && player != g.black {
		return ErrNotInGame
	}

	turn := g.board.Position().Turn()
	if (turn == chess.White && player != g.white) || (turn == chess.Black && player != g.black) {
		return ErrNotYourTurn
	}

	notation := chess.UCINotation{}
	mv, err := notation.Decode(g.board.Position(), move)
	if err != nil {
		return ErrInvalidMove
	}

	if err := g.board.Move(mv); err != nil {
		return ErrInvalidMove
	}

	// var opponent *websocket.Conn
	// if player == g.white {
	// 	opponent = g.black
	// } else {
	// 	opponent = g.white
	// }

	outcome := g.board.Outcome()
	if outcome != chess.NoOutcome {
		g.status = GameStatusCompleted
		g.endTime = time.Now()

		gameOverMsg := OutgoingGameOver{
			Type:    GAME_OVER,
			Outcome: outcome.String(),
			Method:  g.board.Method().String(),
		}

		g.safeSend(g.white, gameOverMsg)
		g.safeSend(g.black, gameOverMsg)
		return nil
	}

	moveMsg := OutgoingMove{Type: MOVE, Move: move}
	g.safeSend(g.white, moveMsg)
	g.safeSend(g.black, moveMsg)

	return nil
}

func (g *Game) HandleDisconnect(player *websocket.Conn) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status != GameStatusInProgress {
		return
	}

	g.status = GameStatusAbandoned
	g.endTime = time.Now()

	var opponent *websocket.Conn
	var outcome string
	if player == g.white {
		opponent = g.black
		outcome = "0-1" // Black wins
	} else {
		opponent = g.white
		outcome = "1-0" // White wins
	}

	if opponent != nil {
		g.safeSend(opponent, OutgoingGameOver{
			Type:    GAME_OVER,
			Outcome: outcome,
			Method:  "Abandonment",
		})
	}

}

func (g *Game) IsPlayer(conn *websocket.Conn) bool {
	g.mu.RLock()
	defer g.mu.RUnlock() // Read lock - readers can read
	return conn == g.white || conn == g.black
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