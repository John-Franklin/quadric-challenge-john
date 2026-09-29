package models

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Errors returned by the user functions.
var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// User is a registered account.
type User struct {
	ID        uint64    `json:"id"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

const userColumns = `id, email, is_admin, created_at`

func (u *User) scanFrom(row scanner, extra ...any) error {
	return row.Scan(append([]any{&u.ID, &u.Email, &u.IsAdmin, &u.CreatedAt}, extra...)...)
}

// CreateUser stores a new user. The email must already be normalized.
func CreateUser(ctx context.Context, db *sql.DB, email, password string, isAdmin bool) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	var u User
	err = u.scanFrom(db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, is_admin) VALUES ($1, $2, $3)
		ON CONFLICT (email) DO NOTHING
		RETURNING `+userColumns,
		email, string(hash), isAdmin,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEmailTaken
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// dummyHash is compared against when the email is unknown, so response timing
// does not reveal which emails are registered.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)

// Authenticate returns the user if the email and password match.
func Authenticate(ctx context.Context, db *sql.DB, email, password string) (*User, error) {
	var u User
	var hash []byte
	err := u.scanFrom(db.QueryRowContext(ctx,
		`SELECT `+userColumns+`, password_hash FROM users WHERE email = $1`, email,
	), &hash)
	if errors.Is(err, sql.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return &u, nil
}

// GetUser returns the user with the given ID, or ErrNotFound.
func GetUser(ctx context.Context, db *sql.DB, id uint64) (*User, error) {
	var u User
	if err := u.scanFrom(db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)); err != nil {
		return nil, notFound(err)
	}
	return &u, nil
}

// ListUsers returns every user, oldest first.
func ListUsers(ctx context.Context, db *sql.DB) ([]User, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+userColumns+` FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := u.scanFrom(rows); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// SetUserAdmin grants or revokes admin access and returns the updated user, or ErrNotFound.
func SetUserAdmin(ctx context.Context, db *sql.DB, id uint64, isAdmin bool) (*User, error) {
	var u User
	if err := u.scanFrom(db.QueryRowContext(ctx, `
		UPDATE users SET is_admin = $2 WHERE id = $1
		RETURNING `+userColumns,
		id, isAdmin,
	)); err != nil {
		return nil, notFound(err)
	}
	return &u, nil
}
