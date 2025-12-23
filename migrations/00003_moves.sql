-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS moves (
    id SERIAL PRIMARY KEY,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    move_number INT NOT NULL,
    move TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(game_id, move_number)
);

CREATE INDEX idx_moves_game_id ON moves(game_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS moves;
-- +goose StatementEnd