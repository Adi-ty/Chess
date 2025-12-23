package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Adi-ty/chess/internal/queue"
	"github.com/Adi-ty/chess/internal/store"
	"github.com/redis/go-redis/v9"
)

type Worker struct {
	rdb *redis.Client
	gameStore store.GameStore
}

func NewWorker(rdb *redis.Client, gameStore store.GameStore) *Worker {
	return &Worker{
		rdb: rdb,
		gameStore: gameStore,
	}
}

func (w *Worker) Start() {
	for {
		// Dequeue and process moves
		result, err := w.rdb.BRPop(context.Background(), 0, "moves_queue").Result()
        if err != nil {
            log.Printf("Worker dequeue error: %v", err)
            time.Sleep(1 * time.Second) // Retry delay
            continue
        }

		var payload queue.MovePayload
        if err := json.Unmarshal([]byte(result[1]), &payload); err != nil {
            log.Printf("Worker unmarshal error: %v", err)
            continue
        }

		if err := w.gameStore.InsertMove(context.Background(), payload); err != nil {
            log.Printf("Worker insert error: %v", err)
            // TODO: re-enqueue or handle failure
        }
	}
}