package database

import (
	"context"
	"time"
)

// ConsumeToken records a single-use token ID. It reports false when the ID was
// already consumed, so a replayed authorization code or rotated refresh token
// is rejected on every replica. Expired records are pruned opportunistically,
// with a grace period so clock skew between app and database cannot reopen a
// token the app still considers unexpired.
func (s *Store) ConsumeToken(ctx context.Context, jti string, expires time.Time) (bool, error) {
	if _, err := s.DB.ExecContext(ctx, "DELETE FROM oauth_consumed_tokens WHERE expires_at < now() - interval '5 minutes'"); err != nil {
		return false, err
	}
	result, err := s.DB.ExecContext(ctx, "INSERT INTO oauth_consumed_tokens(jti,expires_at) VALUES($1,$2) ON CONFLICT DO NOTHING", jti, expires)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
