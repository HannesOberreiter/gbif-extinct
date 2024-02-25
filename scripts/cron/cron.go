package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/HannesOberreiter/gbif-extinct/internal"
	"github.com/HannesOberreiter/gbif-extinct/pkg/gbif"
)

func main() {
	slog.Info("Starting cron")

	var ids []string
	if len(os.Args) > 1 {
		ids = os.Args[1:]
		slog.Info("Fetching observations for specific taxa", "taxa", ids)
	} else {
		ids = internal.GetOutdatedObservations()
		slog.Info("Fetching observations for outdated taxa", "taxa", ids)
	}

	var results [][]gbif.LatestObservation
	for _, id := range ids {
		internal.UpdateLastFetchStatus(id)
		res := gbif.FetchLatest(id)
		if res == nil {
			continue
		}
		results = append(results, res)
	}

	if len(results) == 0 {
		slog.Info("No new observations found")
		return
	}

	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn, err := internal.DB.Conn(ctx)
	if err != nil {
		slog.Error("Failed to create connection", err)
		return
	}
	defer conn.Close()

	internal.SaveObservation(results, conn, ctx)
	cancel()
}
