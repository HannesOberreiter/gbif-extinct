package internal

import (
	"log"
	"log/slog"
	"os"
)

const migrationsDir = "./migrations"

// Helper function to run migration files. Its pretty simple and the migration will always run from the beginning.
// The migrations must be placed in "./migrations" and have the file extension ".sql".
func Migrations() {
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
		log.Printf("Found migration file: %s\n", f.Name())
		file, err := os.ReadFile("migrations/" + f.Name())
		if err != nil {
			log.Fatal(err)
		}
		queries = append(queries, string(file))
	}

	for _, query := range queries {
		log.Printf("Migration of %q\n", query)
		_, err = DB.Exec(query)
		if err != nil {
			log.Printf("%q: %s\n", err, query)
			return
		}
	}
}
