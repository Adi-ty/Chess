package store

import (
	"context"
	"database/sql"
)

type Game struct {
	ID 	  string `json:"id"`
	WhiteUserID string `json:"white_user_id"`
	BlackUserID string `json:"black_user_id"`
	Status string `json:"status"`
	Outcome string `json:"outcome,omitempty"`
	Method string `json:"method,omitempty"`
	StartedAt string `json:"started_at"`
	EndedAt string `json:"ended_at,omitempty"`
}

type GameStore interface {
	CreateGame(ctx context.Context, game *Game) (*Game, error)
	// GetGameByUserID(ctx context.Context, id string) (*Game, error)
	UpdateGameStatus(ctx context.Context, id string, status string, outcome string, method string, endedAt string) error
}

type PostgresGameStore struct {
	db *sql.DB
}

func NewPostgresGameStore(db *sql.DB) *PostgresGameStore {
	return &PostgresGameStore{db: db}
}

func (s *PostgresGameStore) CreateGame(ctx context.Context, game *Game) (*Game, error) {
	var g Game
	
	query := `
		INSERT INTO games (id, white_user_id, black_user_id, status, started_at, ended_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, white_user_id, black_user_id, status, started_at, ended_at
	`

	err := s.db.QueryRowContext(ctx, query,
		game.ID,
		game.WhiteUserID,
		game.BlackUserID,
		game.Status,
		game.StartedAt,
		sql.NullString{String: game.EndedAt, Valid: game.EndedAt != ""},
	).Scan(&g.ID, &g.WhiteUserID, &g.BlackUserID, &g.Status, &g.StartedAt, &g.EndedAt)
	
	if err != nil {
		return nil, err
	}

	return &g, nil
}

// func (s *PostgresGameStore) GetGameByUserID(ctx context.Context, id string) (*Game, error) {
// 	var g Game

// 	query := `
//         SELECT id, white_user_id, black_user_id, status, started_at, ended_at
//         FROM games
//         WHERE (white_user_id = $1 OR black_user_id = $1) AND status = 'in_progress'
//         ORDER BY started_at DESC
//         LIMIT 1
//     `

// 	row := s.db.QueryRowContext(ctx, query, id)
// 	err := row.Scan(&g.ID, &g.WhiteUserID, &g.BlackUserID, &g.Status, &g.StartedAt, &g.EndedAt)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	return &g, nil
// }

func (s *PostgresGameStore) UpdateGameStatus(ctx context.Context, id string, status string, outcome string, method string, endedAt string) error {
	query := `
		UPDATE games
		SET status = $1, outcome = $2, method = $3, ended_at = $4
		WHERE id = $5
	`

	_, err := s.db.ExecContext(ctx, query, status, outcome, method, endedAt, id)

	return err
}