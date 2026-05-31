package store

import (
	"context"
	"errors"
	"fmt"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/jackc/pgx/v5"
)

// ListUsers returns all demo users.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, username FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// GetUserByID loads a user or returns not found.
func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `SELECT id::text, username FROM users WHERE id = $1`, id).Scan(&u.ID, &u.Username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.New(apperrors.CodeNotFound, "user not found")
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser inserts a new simulated user with zero balances for known assets.
func (s *Store) CreateUser(ctx context.Context, username string) (*User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var u User
	err = tx.QueryRow(ctx,
		`INSERT INTO users (username) VALUES ($1) RETURNING id::text, username`,
		username,
	).Scan(&u.ID, &u.Username)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO balances (user_id, asset_id, available, frozen)
		SELECT $1, id, 0, 0 FROM assets
		ON CONFLICT DO NOTHING`, u.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &u, nil
}
