package gamemanager

import (
	"errors"
	"time"

	"github.com/gorilla/websocket"
	"github.com/notnil/chess"
)

type Game struct {
	white *websocket.Conn
	black *websocket.Conn
	board *chess.Game
	startTime time.Time
}

func StartNewGame(player1, player2 *websocket.Conn) *Game {
	game := &Game{
		white: player1,
		black: player2,
		board: chess.NewGame(),
		startTime: time.Now(),
	}

	player1.WriteJSON(map[string]string{"type": "game_start", "color": "white"})
    player2.WriteJSON(map[string]string{"type": "game_start", "color": "black"})

	return game
}

func (g *Game) MakeMove(player *websocket.Conn, move string) error {
	turn := g.board.Position().Turn()
	if (turn == chess.White && player != g.white) || (turn == chess.Black && player != g.black) {
		return errors.New("not your turn buddy")
	}
	
	if err := g.board.MoveStr(move); err != nil {
		return err
	}

	var opponent *websocket.Conn
    if player == g.white {
        opponent = g.black
    } else {
        opponent = g.white
    }

	outcome := g.board.Outcome()
	if outcome != chess.NoOutcome {
		gameOverMsg := OutgoingGameOver{
            Type:    GAME_OVER,
            Outcome: outcome.String(),
            Method:  g.board.Method().String(),
        }
		
		g.white.WriteJSON(gameOverMsg)
		g.black.WriteJSON(gameOverMsg)
		return nil
	}

	moveMsg := OutgoingMove{Type: MOVE, Move: move}
	opponent.WriteJSON(moveMsg)
	
	return nil
}