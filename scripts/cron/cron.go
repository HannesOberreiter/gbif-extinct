package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/HannesOberreiter/gbif-extinct/internal"
	"github.com/HannesOberreiter/gbif-extinct/pkg"
)

func main() {
	slog.Info("Starting cron")

	ids := internal.GetOutdatedObservations()
	var results [][]pkg.LatestObservation
	for _, id := range ids {
		internal.UpdateLastFetchStatus(id)
		res := pkg.FetchLatest(id)
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
