package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/agent-guide/caddy-llm/llm/config"
	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

// Store is the SQLite implementation of config.Store.
type Store struct {
	db *sql.DB
}

// New opens (or creates) a SQLite config database.
func New(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite config: open: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS config (
			key        TEXT PRIMARY KEY,
			value      TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func (s *Store) Get(ctx context.Context, key string, dest any) error {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM config WHERE key = ?`, key).Scan(&raw)
	if err == sql.ErrNoRows {
		return fmt.Errorf("config: key %q not found", key)
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), dest)
}

func (s *Store) Set(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, key, string(data))
	return err
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM config WHERE key = ?`, key)
	return err
}

func (s *Store) List(ctx context.Context, prefix string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key FROM config WHERE key LIKE ?`, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *Store) Tx(ctx context.Context, fn func(tx config.Store) error) error {
	sqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txStore := &txStore{tx: sqlTx}
	if err := fn(txStore); err != nil {
		sqlTx.Rollback()
		return err
	}
	return sqlTx.Commit()
}

func (s *Store) Close() error {
	return s.db.Close()
}

// txStore wraps a sql.Tx to implement config.Store within a transaction.
type txStore struct {
	tx *sql.Tx
}

func (t *txStore) Get(ctx context.Context, key string, dest any) error {
	var raw string
	err := t.tx.QueryRowContext(ctx, `SELECT value FROM config WHERE key = ?`, key).Scan(&raw)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), dest)
}

func (t *txStore) Set(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(ctx, `
		INSERT INTO config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, key, string(data))
	return err
}

func (t *txStore) Delete(ctx context.Context, key string) error {
	_, err := t.tx.ExecContext(ctx, `DELETE FROM config WHERE key = ?`, key)
	return err
}

func (t *txStore) List(ctx context.Context, prefix string) ([]string, error) {
	rows, err := t.tx.QueryContext(ctx, `SELECT key FROM config WHERE key LIKE ?`, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (t *txStore) Tx(ctx context.Context, fn func(tx config.Store) error) error {
	return fn(t)
}
