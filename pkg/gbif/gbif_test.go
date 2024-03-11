package gbif

import (
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
