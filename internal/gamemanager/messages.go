package gamemanager

type Message struct {
    Type string `json:"type"`
}

const INIT_GAME = "init_game"