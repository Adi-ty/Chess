package gamemanager

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type GameManager struct {
	games []*Game
	pendingUser *websocket.Conn
	users []*websocket.Conn
}

func NewGameManager() *GameManager {
	return &GameManager{
		games: []*Game{},
		pendingUser: nil,
		users: []*websocket.Conn{},
	}
}

func (gm *GameManager) AddUser(conn *websocket.Conn) {
	gm.users = append(gm.users, conn)
}

func (gm *GameManager) RemoveUser(conn *websocket.Conn) {
	gm.users = filterOutConn(gm.users, conn)

}


func (gm *GameManager) AddHandler(conn *websocket.Conn) {
	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		
		var message IncomingMessage
		if err := json.Unmarshal(rawMsg, &message); err != nil {
			conn.WriteJSON(OutgoingError{Type: ERROR, Message: "invalid message format"})
			continue
		}
		
		switch message.Type {
		case INIT_GAME:
			if gm.pendingUser != nil {
				game := StartNewGame(gm.pendingUser, conn)
				gm.games = append(gm.games, game)
				gm.pendingUser = nil
			} else {
				gm.pendingUser = conn
			}
		case MOVE:
			game := findGameByConn(gm.games, conn)
			if game != nil {
				game.MakeMove(conn, message.Move)
			}
		}
	}
}

func filterOutConn(conns []*websocket.Conn, target *websocket.Conn) []*websocket.Conn {
	result := make([]*websocket.Conn, 0, len(conns))
	for _, conn := range conns {
		if conn != target {
			result = append(result, conn)
		}
	}
	return result
}

func findGameByConn(games []*Game, conn *websocket.Conn) *Game {
	for _, game := range games {
		if game.white == conn || game.black == conn {
			return game
		}
	}
	return nil
}
	