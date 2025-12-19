package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
    ErrUserNotFound = errors.New("user not found")
)

type User struct {
    ID          string    `json:"id"`
    Email       string    `json:"email"`
    DisplayName string    `json:"display_name"`
    AvatarURL   string    `json:"avatar_url,omitempty"`
    Provider    string    `json:"provider"`
    ProviderID  string    `json:"provider_id"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type PostgresUserStore struct {
    db *sql.DB
}

type UserStore interface {
	CreateOrUpdate(ctx context.Context, user *User) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
}

func NewPostgresUserStore(db *sql.DB) *PostgresUserStore {
	return &PostgresUserStore{db: db}
}

func (s *PostgresUserStore) CreateOrUpdate(ctx context.Context, user *User) (*User, error) {
	var u User

	query := `
        INSERT INTO users (email, display_name, avatar_url, provider, provider_id)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (provider, provider_id) DO UPDATE SET
            email = EXCLUDED.email,
            display_name = EXCLUDED.display_name,
            avatar_url = EXCLUDED.avatar_url,
            updated_at = NOW()
        RETURNING id, email, display_name, avatar_url, provider, provider_id, created_at, updated_at
    `

	err := s.db.QueryRowContext(ctx, query,
        user.Email,
        user.DisplayName,
        user.AvatarURL,
        user.Provider,
        user.ProviderID,
    ).Scan(
        &u.ID,
        &u.Email,
        &u.DisplayName,
        &u.AvatarURL,
        &u.Provider,
        &u.ProviderID,
        &u.CreatedAt,
        &u.UpdatedAt,
    )

    if err != nil {
        return nil, err
    }

    return &u, nil
}

func (s *PostgresUserStore) GetUserByID(ctx context.Context, id string) (*User, error) {
    query := `
        SELECT id, email, display_name, avatar_url, provider, provider_id, created_at, updated_at
        FROM users WHERE id = $1
    `

    var u User
    err := s.db.QueryRowContext(ctx, query, id).Scan(
        &u.ID,
        &u.Email,
        &u.DisplayName,
        &u.AvatarURL,
        &u.Provider,
        &u.ProviderID,
        &u.CreatedAt,
        &u.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, ErrUserNotFound
    }
    if err != nil {
        return nil, err
    }

    return &u, nil
}