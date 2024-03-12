package queries

import (
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

var DemoObservation = []string{DemoTaxa[0], "123456", "'1989-01-05'", "'1989-01-05'", "'AT'"}

// Demo data for testing which is a synonym
var DemoSyn = []string{
	"8071112", "4492208", "'Urocerus gigas'", "'Ichneumon gigas'", "'Animalia'", "'Arthropoda'", "'Insecta'", "'Hymenoptera'", "'Siricidae'", "'Urocerus'", "true",
}

func TestQuery(t *testing.T) {
	loadDemo()
	q := Query{
		ORDER_BY:      "Date",
		ORDER_DIR:     "ASC",
		HIDE_SYNONYMS: false,
	}

	counts := GetCounts(internal.DB, q)
	if counts.TaxaCount != 1 {
		t.Errorf("got %d, wanted %d", counts.TaxaCount, 1)
	}

	if counts.ObservationCount != 1 {
		t.Errorf("got %d, wanted %d", counts.ObservationCount, 1)
	}

	table := GetTableData(internal.DB, q)
	if len(table) != 2 {
		t.Errorf("got %d, wanted %d", len(table), 2)
	}

	q.HIDE_SYNONYMS = true
	table = GetTableData(internal.DB, q)
	if len(table) != 1 {
		t.Errorf("got %d, wanted %d", len(table), 1)
	}

	if table[0].TaxonID != DemoTaxa[0] {
		t.Errorf("got %s, wanted %s", table[0].TaxonID, DemoTaxa[0])
	}

}

func TestNewQuery(t *testing.T) {
	q := NewQuery(nil)
	if q.ORDER_BY != "date" {
		t.Errorf("got %s, wanted %s", q.ORDER_BY, "date")
	}

	var order_by = "test"
	var payload = struct {
		ORDER_BY *string
	}{
		ORDER_BY: &order_by,
	}

	q = NewQuery(payload)
	if q.ORDER_BY != "test" {
		t.Errorf("got %s, wanted %s", q.ORDER_BY, "test")
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

	_, err = internal.DB.Exec(`
		INSERT OR REPLACE INTO observations
		(TaxonID, ObservationID, ObservationDateOriginal, ObservationDate, CountryCode)
		VALUES (` + strings.Join(DemoObservation, ",") + ")")
	if err != nil {
		slog.Error("Database error", err)
		log.Fatal(err)
	}
}
