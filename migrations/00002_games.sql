-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    white_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    black_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress',
    outcome VARCHAR(10),
    method VARCHAR(50),
    pgn TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT valid_status CHECK (status IN ('in_progress', 'completed', 'abandoned'))
);

CREATE INDEX idx_games_white_user ON games(white_user_id);
CREATE INDEX idx_games_black_user ON games(black_user_id);
CREATE INDEX idx_games_status ON games(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS games;
-- +goose StatementEnd