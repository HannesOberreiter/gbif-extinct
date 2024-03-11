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
	internal.Load()

	gbif.UpdateConfig(gbif.Config{UserAgentPrefix: internal.Config.UserAgentPrefix})

	var ids []string
	if len(os.Args) > 1 {
		ids = os.Args[1:]
		slog.Info("Fetching observations for specific taxa", "taxa", ids)
	} else {
		ids = gbif.GetOutdatedObservations(internal.DB)
		slog.Info("Fetching observations for outdated taxa", "taxa", ids)
	}

	var results [][]gbif.LatestObservation
	for _, id := range ids {
		gbif.UpdateLastFetchStatus(internal.DB, id)
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

	gbif.SaveObservation(results, conn, ctx)
	cancel()
}
