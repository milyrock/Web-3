package repository

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed schema.sql
var schema string

type Repository struct {
	db *sql.DB
}

func Open(ctx context.Context, databaseURL string) (*Repository, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	r := &Repository{db: db}
	if err = db.PingContext(ctx); err == nil {
		err = r.init(ctx)
	}
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	return r, nil
}

func (r *Repository) Close() error { return r.db.Close() }

func (r *Repository) init(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(49103)"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)"); err != nil {
		return err
	}
	var version int
	if err := tx.QueryRowContext(ctx, "SELECT coalesce(max(version),0) FROM schema_migrations").Scan(&version); err != nil {
		return err
	}
	if version == 0 {
		if _, err := tx.ExecContext(ctx, schema); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version) VALUES(1),(2)"); err != nil {
			return err
		}
	} else if version == 1 {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE feedback ALTER COLUMN user_id DROP NOT NULL;
			ALTER TABLE feedback ADD COLUMN name TEXT NOT NULL DEFAULT 'Гость';
			ALTER TABLE feedback ADD COLUMN email TEXT NOT NULL DEFAULT '';
			ALTER TABLE feedback ADD COLUMN consent_at TIMESTAMPTZ;
			ALTER TABLE feedback ALTER COLUMN consent_at SET DEFAULT CURRENT_TIMESTAMP;
			UPDATE feedback f SET name=u.name,email=u.email FROM users u WHERE f.user_id=u.id;
			INSERT INTO schema_migrations(version) VALUES(2)`); err != nil {
			return err
		}
	}
	if version < 3 {
		if _, err := tx.ExecContext(ctx, `CREATE TABLE auth_users (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			name TEXT NOT NULL CHECK(char_length(trim(name)) BETWEEN 2 AND 100),
			email TEXT NOT NULL UNIQUE CHECK(email=lower(email)),
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('user','admin'))
		);
		CREATE TABLE auth_sessions (
			token_hash TEXT PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
			expires_at TIMESTAMPTZ NOT NULL
		);
		CREATE INDEX auth_sessions_expiry ON auth_sessions(expires_at);
		INSERT INTO schema_migrations(version) VALUES(3)`); err != nil {
			return err
		}
	}
	if version < 4 {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE products ADD COLUMN stock INTEGER NOT NULL DEFAULT 10 CHECK(stock >= 0);
			INSERT INTO schema_migrations(version) VALUES(4)`); err != nil {
			return err
		}
	}
	return tx.Commit()
}
