package internal

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"
)

var migrationsDir = "./migrations"

// Helper function to run migration files. Its pretty simple and the migration will always run from the beginning.
// The migrations must be placed in "./migrations" and have the file extension ".sql".
func Migrations(db *sql.DB, migrationsPath string) {
	if migrationsPath != "" {
		migrationsDir = migrationsPath + "/migrations"
	}
	slog.Debug("Running migrations")
	fs, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatal(err)
	}
	var queries []string
	for _, f := range fs {
		if f.IsDir() {
			continue
		}
		if f.Name()[len(f.Name())-4:] != ".sql" {
			continue
		}
		slog.Info("Found migration file", "file", f.Name())
		file, err := os.ReadFile(migrationsDir + "/" + f.Name())
		if err != nil {
			log.Fatal(err)
		}
		queries = append(queries, string(file))
	}
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		slog.Error("Failed to create migration connection", err)
		return
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	defer tx.Rollback()
	if err != nil {
		slog.Error("Failed to start migration transaction", err)
		return
	}

	for _, query := range queries {
		_, err = tx.Exec(query)
		if err != nil {
			slog.Error("Failed to run migration", "error", err, "query", query)
			return
		}
	}
	err = tx.Commit()
	if err != nil {
		slog.Error("Failed to commit migration", "error", err)
		return
	}
	conn.Close()
	slog.Info("Migrations ran successfully")
}
