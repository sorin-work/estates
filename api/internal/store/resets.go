package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrResetInvalid means a reset token is unknown, used or expired.
var ErrResetInvalid = errors.New("reset link is invalid or has expired")

// CreatePasswordReset stores the hash of a new reset token for a user and
// drops links that expired a day ago or more, so the table stays small.
func (s *Store) CreatePasswordReset(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM password_resets WHERE expires_at < now() - interval '1 day'`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
			userID, tokenHash, expiresAt)
		return err
	})
}

// ResetPassword uses a reset token: if it is known, unused and not expired
// at now, the user's password becomes passwordHash, their token version
// goes up (which signs out every session) and every other open link of
// theirs is used up too. It all happens in one transaction, so a token
// works once even when two requests race.
func (s *Store) ResetPassword(ctx context.Context, tokenHash []byte, passwordHash string, now time.Time) (User, error) {
	var u User
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var userID string
		err := tx.QueryRow(ctx, `
			UPDATE password_resets SET used_at = $2
			 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2
			RETURNING user_id`, tokenHash, now).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResetInvalid
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE password_resets SET used_at = $2 WHERE user_id = $1 AND used_at IS NULL`, userID, now); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			UPDATE users SET password_hash = $2, token_version = token_version + 1, updated_at = now()
			 WHERE id = $1
			RETURNING `+userColumns, userID, passwordHash)
		u, err = collectOne[User](rows, err)
		return err
	})
	return u, err
}
