package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"developer.dev/web-3/internal/model"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repository) CreateUser(ctx context.Context, user model.User) (model.User, error) {
	err := sq.Insert("auth_users").Columns("name", "email", "password_hash", "role").
		Values(user.Name, user.Email, user.PasswordHash, user.Role).Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar).RunWith(r.db).QueryRowContext(ctx).Scan(&user.ID)
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return model.User{}, model.ErrEmailExists
	}
	return user, err
}

func (r *Repository) UserByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User
	err := sq.Select("id", "name", "email", "role", "password_hash").From("auth_users").
		Where(sq.Eq{"email": email}).PlaceholderFormat(sq.Dollar).RunWith(r.db).
		QueryRowContext(ctx).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return user, model.ErrNotFound
	}
	return user, err
}

func (r *Repository) CreateSession(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error {
	_, err := sq.Insert("auth_sessions").Columns("token_hash", "user_id", "expires_at").
		Values(tokenHash, userID, expiresAt).PlaceholderFormat(sq.Dollar).RunWith(r.db).ExecContext(ctx)
	return err
}

func (r *Repository) Session(ctx context.Context, tokenHash string) (model.Session, error) {
	var session model.Session
	err := sq.Select("u.id", "u.name", "u.email", "u.role", "s.expires_at").
		From("auth_sessions s").Join("auth_users u ON u.id=s.user_id").
		Where(sq.Eq{"s.token_hash": tokenHash}).PlaceholderFormat(sq.Dollar).RunWith(r.db).
		QueryRowContext(ctx).Scan(&session.User.ID, &session.User.Name, &session.User.Email, &session.User.Role, &session.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return session, model.ErrNotFound
	}
	return session, err
}

func (r *Repository) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := sq.Delete("auth_sessions").Where(sq.Eq{"token_hash": tokenHash}).
		PlaceholderFormat(sq.Dollar).RunWith(r.db).ExecContext(ctx)
	return err
}
