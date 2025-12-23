package queue

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type MovePayload struct {
	GameID string `json:"game_id"`
	UserID string `json:"user_id"`
	MoveNumber int    `json:"move_number"`
	Move     string `json:"move"`
	CreatedAt float64 `json:"created_at"`
}

func EnqueueMove(redisClient *redis.Client, payload MovePayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return redisClient.LPush(context.Background(), "moves_queue", jsonData).Err()
}