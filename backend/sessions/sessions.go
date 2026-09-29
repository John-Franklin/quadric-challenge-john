// Package sessions stores login sessions in Redis, keyed by a hash of an opaque random token.
package sessions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultAddr = "localhost:6379"

// TTL is how long a session lasts after login.
const TTL = 7 * 24 * time.Hour

// ErrNoSession is returned for unknown or expired tokens.
var ErrNoSession = errors.New("session not found")

// Store is a Redis-backed session store.
type Store struct {
	rdb *redis.Client
}

// Connect opens a Redis client and verifies it with a ping.
// The address is read from REDIS_ADDR, falling back to the compose.yml default.
func Connect(ctx context.Context) (*Store, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, err
	}
	return &Store{rdb: rdb}, nil
}

// Close closes the Redis client.
func (s *Store) Close() error {
	return s.rdb.Close()
}

// Only the token's hash is stored, so a leaked Redis snapshot cannot be replayed as cookies.
func key(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "session:" + hex.EncodeToString(sum[:])
}

// Create starts a session for the user and returns the token to hand to the client.
func (s *Store) Create(ctx context.Context, userID uint64) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	if err := s.rdb.Set(ctx, key(token), userID, TTL).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// UserID returns the user the token belongs to, or ErrNoSession if it is unknown or expired.
func (s *Store) UserID(ctx context.Context, token string) (uint64, error) {
	val, err := s.rdb.Get(ctx, key(token)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrNoSession
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(val, 10, 64)
}

// Delete ends the session for token; unknown tokens are ignored.
func (s *Store) Delete(ctx context.Context, token string) error {
	return s.rdb.Del(ctx, key(token)).Err()
}
