package main

import (
	"log/slog"
	"os"

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

	var results = &[][]gbif.LatestObservation{}
	for _, id := range ids {
		gbif.UpdateLastFetchStatus(internal.DB, id)
		res := gbif.FetchLatest(id)
		if res == nil {
			continue
		}
		*results = append(*results, *res)
	}

	if len(*results) == 0 {
		slog.Info("No new observations found")
		return
	}

	gbif.SaveObservation(results, internal.DB)
}
