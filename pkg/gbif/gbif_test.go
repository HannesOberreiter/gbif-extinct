package gbif

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"strings"
	"testing"

	"github.com/HannesOberreiter/gbif-extinct/internal"
)

// Demo data for testing, it is no synonym
var DemoTaxa = []string{
	"4492208", "4492208", "'Urocerus gigas'", "'Animalia'", "'Arthropoda'", "'Insecta'", "'Hymenoptera'", "'Siricidae'", "'Urocerus'",
}

// Demo data for testing which is a synonym
var DemoSyn = []string{
	"8071112", "4492208", "'Urocerus gigas'", "'Ichneumon gigas'", "'Animalia'", "'Arthropoda'", "'Insecta'", "'Hymenoptera'", "'Siricidae'", "'Urocerus'", "true",
}

func TestUpdateConfig(t *testing.T) {
	UpdateConfig(Config{UserAgentPrefix: "test"})
}

func TestFetchLatest(t *testing.T) {
	loadDemo()

	res := FetchLatest(DemoTaxa[0])
	if res == nil {
		t.Errorf("got %v, wanted %v", res, "not nil")
	}
	if len(res) == 0 {
		t.Errorf("got %d, wanted < %d", len(res), 1)
	}
	if res[0].TaxonID != DemoTaxa[0] {
		t.Errorf("got %s, wanted %s", res[0].TaxonID, DemoTaxa[0])
	}

	res = FetchLatest("123456")
	if res != nil {
		t.Errorf("got %v, wanted %v", res, nil)
	}
}

func TestSaveObservations(t *testing.T) {
	loadDemo()
	observation := LatestObservation{
		TaxonID:                 DemoTaxa[0],
		ObservationID:           "123456",
		ObservationOriginalDate: "1989-01-05",
		ObservationDate:         "1989-01-05",
		CountryCode:             "AT",
	}

	var observations [][]LatestObservation
	observations = append(observations, []LatestObservation{observation})

	ctx := context.Background()
	conn, err := internal.DB.Conn(ctx)
	if err != nil {
		slog.Error("Failed to create connection", err)
	}
	defer conn.Close()

	SaveObservation(observations, conn, ctx)

	var count int
	err = internal.DB.QueryRow("SELECT COUNT(*) FROM observations WHERE TaxonID = ?", DemoTaxa[0]).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	if count != 1 {
		t.Errorf("got %d, wanted %d", count, 1)
	}
}

func TestGetOutdatedObservations(t *testing.T) {
	loadDemo()
	want := GetOutdatedObservations(internal.DB)
	if len(want) != 1 {
		t.Errorf("got %d, wanted %d", len(want), 1)
	}
	if want[0] != DemoTaxa[0] {
		t.Errorf("got %s, wanted %s", want[0], DemoTaxa[0])
	}
}

func TestUpdateLastFetchStatus(t *testing.T) {
	loadDemo()
	var lastFetch sql.NullTime

	/* If not set for a taxa return null */
	err := internal.DB.QueryRow("SELECT LastFetch FROM taxa WHERE TaxonID = ?", DemoTaxa[0]).Scan(&lastFetch)
	if err != nil {
		log.Fatal(err)
	}
	if lastFetch.Valid {
		t.Errorf("got %v, wanted %v", lastFetch.Valid, false)
	}

	/* Update last fetch status and it should return one */
	UpdateLastFetchStatus(internal.DB, DemoTaxa[0])

	err = internal.DB.QueryRow("SELECT LastFetch FROM taxa WHERE TaxonID = ?", DemoTaxa[0]).Scan(&lastFetch)
	if err != nil {
		log.Fatal(err)
	}
	if !lastFetch.Valid {
		t.Errorf("got %v, wanted %v", lastFetch.Valid, true)
	}
	if lastFetch.Time.IsZero() {
		t.Errorf("got %v, wanted %v", lastFetch.Time.IsZero(), false)
	}
}

func TestGetSynonymID(t *testing.T) {
	loadDemo()

	/* Actual synonym */
	want, err := GetSynonymID(internal.DB, DemoSyn[0])
	if err != nil {
		t.Errorf("got %v, wanted %v", err, nil)
	}
	if want != DemoSyn[1] {
		t.Errorf("got %s, wanted %s", want, DemoSyn[1])
	}

	/* If no synonym return itself */
	want, err = GetSynonymID(internal.DB, DemoTaxa[0])
	if err != nil {
		t.Errorf("got %v, wanted %v", err, nil)
	}
	if want != DemoTaxa[0] {
		t.Errorf("got %s, wanted %s", want, DemoTaxa[0])
	}

	/* If no taxon in database return error */
	_, err = GetSynonymID(internal.DB, "123456")
	if err == nil {
		t.Errorf("got %v, wanted %v", err, "error")
	}
}

// Helper to setup memory database and data
func loadDemo() {
	slog.SetLogLoggerLevel(slog.LevelError)
	internal.Load()
	internal.Migrations(internal.DB, internal.Config.ROOT)

	_, err := internal.DB.Exec(`
		INSERT OR REPLACE INTO taxa
		(TaxonID, SynonymID, ScientificName, TaxonKingdom, TaxonPhylum, TaxonClass, TaxonOrder, TaxonFamily, TaxonGenus)
		VALUES (` + strings.Join(DemoTaxa, ",") + ")")
	if err != nil {
		slog.Error("Database error", err)
		log.Fatal(err)
	}
	_, err = internal.DB.Exec(`
		INSERT OR REPLACE INTO taxa
		(TaxonID, SynonymID, SynonymName, ScientificName, TaxonKingdom, TaxonPhylum, TaxonClass, TaxonOrder, TaxonFamily, TaxonGenus, isSynonym)
		VALUES  (` + strings.Join(DemoSyn, ",") + ")")
	if err != nil {
		slog.Error("Database error", err)
		log.Fatal(err)
	}
}
