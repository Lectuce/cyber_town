package game

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS worlds (
	id TEXT PRIMARY KEY NOT NULL,
	game_time TEXT NOT NULL,
	revision INTEGER NOT NULL CHECK (revision >= 1)
);

CREATE TABLE IF NOT EXISTS scenes (
	world_id TEXT NOT NULL,
	id TEXT NOT NULL,
	name TEXT NOT NULL,

	PRIMARY KEY (world_id, id),
	FOREIGN KEY (world_id) REFERENCES worlds(id)
);

CREATE TABLE IF NOT EXISTS players (
	world_id TEXT NOT NULL,
	id TEXT NOT NULL,
	name TEXT NOT NULL,
	scene_id TEXT NOT NULL,
	status TEXT NOT NULL,
	position_x REAL NOT NULL,
	position_y REAL NOT NULL,
	floor INTEGER NOT NULL CHECK (floor BETWEEN 0 AND 3),
	revision INTEGER NOT NULL CHECK (revision >= 1),

	PRIMARY KEY (world_id, id),
	FOREIGN KEY (world_id, scene_id)
		REFERENCES scenes(world_id, id)
);

CREATE TABLE IF NOT EXISTS npcs (
	world_id TEXT NOT NULL,
	id TEXT NOT NULL,
	name TEXT NOT NULL,
	role TEXT NOT NULL,
	persona TEXT NOT NULL,
	background TEXT NOT NULL,
	scene_id TEXT NOT NULL,
	current_activity TEXT NOT NULL,
	position_x REAL NOT NULL,
	position_y REAL NOT NULL,
	floor INTEGER NOT NULL CHECK (floor BETWEEN 0 AND 3),
	revision INTEGER NOT NULL CHECK (revision >= 1),

	PRIMARY KEY (world_id, id),
	FOREIGN KEY (world_id, scene_id)
		REFERENCES scenes(world_id, id)
);
`

func OpenDatabase(ctx context.Context, path string) (*sql.DB, error) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	dsn := filepath.ToSlash(path) +
		"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect database: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("begin schema transaction: %w", err)
	}

	if _, err := tx.ExecContext(ctx, schemaSQL); err != nil {
		tx.Rollback()
		db.Close()
		return nil, fmt.Errorf("create game tables: %w", err)
	}

	if err := tx.Commit(); err != nil {
		db.Close()
		return nil, fmt.Errorf("commit game tables: %w", err)
	}

	return db, nil
}
