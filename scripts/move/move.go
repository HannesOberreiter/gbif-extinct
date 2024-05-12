/* This is only a one-time script. It moved the old duckDB database to sqllite */
package main

import (
	"database/sql"
	"log"
	"log/slog"
	"strings"

	"github.com/HannesOberreiter/gbif-extinct/internal"
	_ "github.com/marcboeker/go-duckdb"
)

var (
	DUCK *sql.DB
)

func main() {
	loadDuck()
	internal.Load()
	move("observations")
	move("taxa")
}

func loadDuck() {
	var err error
	var dbPath = "./db/duck.db"

	DUCK, err = sql.Open("duckdb", dbPath)
	if err != nil {
		slog.Debug("Failed to connect to database.", "path", dbPath)
		log.Fatal(err)
	}

	slog.Info("Connected to database.", "path", dbPath)
}

func move(table string) {
	transaction, err := internal.DB.Begin()
	if err != nil {
		slog.Error("Failed to start transaction", err)
		return
	}
	defer transaction.Rollback()

	rows, err := DUCK.Query("SELECT * FROM " + table)
	if err != nil {
		slog.Error("Failed to fetch "+table, err)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		log.Fatal(err)
	}
	query := "INSERT OR REPLACE INTO " + table + " (" + strings.Join(columns, ", ") + ") VALUES (?" + strings.Repeat(", ?", len(columns)-1) + ")"
	stmt, err := transaction.Prepare(query)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	count := 0

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := 0; i < len(columns); i++ {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Fatal(err)
		}

		if _, err := stmt.Exec(values...); err != nil {
			log.Fatal(err)
		}
		count++
		if count%100_000 == 0 {
			slog.Info("Moved "+table, "count", count)
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	if err := transaction.Commit(); err != nil {
		slog.Error("Failed to commit transaction", err)
	}
	slog.Info("Finish moved "+table, "count", count)
}
