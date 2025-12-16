package gamemanager

type IncomingMessage struct {
	Type string `json:"type"`
	Move string `json:"move,omitempty"`
}

type OutgoingMove struct {
	Type string `json:"type"`
	Move string `json:"move"`
}

type OutgoingGameOver struct {
	Type    string `json:"type"`
	Outcome string `json:"outcome"`
	Method  string `json:"method"`
}

type OutgoingError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type OutgoingWaiting struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

const (
	INIT_GAME = "init_game"
	MOVE      = "move"
	GAME_OVER = "game_over"
	ERROR     = "error"
	WAITING   = "waiting"
)
