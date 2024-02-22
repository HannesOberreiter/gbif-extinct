// Init logic to get the local database ready.
// Loading gbif backbone taxonomy into the local database.
// To download the taxonomy see https://hosted-datasets.gbif.org/datasets/backbone/
package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/HannesOberreiter/gbif-extinct/internal"
)

var conn *sql.Conn

// Main function to start the mutation, it will populate the taxa table with data from the gbif backbone taxonomy. You need to have the TAXON_BACKBONE_PATH set in the .env file. The file can be downloaded from https://hosted-datasets.gbif.org/datasets/backbone/.
// You can run this script with `go run scripts/mutate/mutate.go`.
func main() {
	slog.Info("Starting mutation")
	var err error
	conn, err = internal.DB.Conn(context.Background())
	if err != nil {
		slog.Error("Failed to connect to database", err)
		return
	}
	defer conn.Close()
	populateTaxa()
}

// Populate taxon table with data from gbif backbone taxonomy
//
//	 <core encoding="UTF-8" fieldsTerminatedBy="\t" linesTerminatedBy="\n" fieldsEnclosedBy="" ignoreHeaderLines="1" rowType="http://rs.tdwg.org/dwc/terms/Taxon">
//	  <files>
//	    <location>Taxon.tsv</location>
//	  </files>
//	  <id index="0" />
//	  <field index="1" term="http://rs.tdwg.org/dwc/terms/datasetID"/>
//	  <field index="2" term="http://rs.tdwg.org/dwc/terms/parentNameUsageID"/>
//	  <field index="3" term="http://rs.tdwg.org/dwc/terms/acceptedNameUsageID"/>
//	  <field index="4" term="http://rs.tdwg.org/dwc/terms/originalNameUsageID"/>
//	  <field index="5" term="http://rs.tdwg.org/dwc/terms/scientificName"/>
//	  <field index="6" term="http://rs.tdwg.org/dwc/terms/scientificNameAuthorship"/>
//	  <field index="7" term="http://rs.gbif.org/terms/1.0/canonicalName"/>
//	  <field index="8" term="http://rs.tdwg.org/dwc/terms/genericName"/>
//	  <field index="9" term="http://rs.tdwg.org/dwc/terms/specificEpithet"/>
//	  <field index="10" term="http://rs.tdwg.org/dwc/terms/infraspecificEpithet"/>
//	  <field index="11" term="http://rs.tdwg.org/dwc/terms/taxonRank"/>
//	  <field index="12" term="http://rs.tdwg.org/dwc/terms/nameAccordingTo"/>
//	  <field index="13" term="http://rs.tdwg.org/dwc/terms/namePublishedIn"/>
//	  <field index="14" term="http://rs.tdwg.org/dwc/terms/taxonomicStatus"/>
//	  <field index="15" term="http://rs.tdwg.org/dwc/terms/nomenclaturalStatus"/>
//	  <field index="16" term="http://rs.tdwg.org/dwc/terms/taxonRemarks"/>
//	  <field index="17" term="http://rs.tdwg.org/dwc/terms/kingdom"/>
//	  <field index="18" term="http://rs.tdwg.org/dwc/terms/phylum"/>
//	  <field index="19" term="http://rs.tdwg.org/dwc/terms/class"/>
//	  <field index="20" term="http://rs.tdwg.org/dwc/terms/order"/>
//	  <field index="21" term="http://rs.tdwg.org/dwc/terms/family"/>
//	  <field index="22" term="http://rs.tdwg.org/dwc/terms/genus"/>
//	</core>
func populateTaxa() {
	slog.Info("Populating taxa table", "file", internal.Config.TaxonBackbonePath)
	file, err := os.Open(internal.Config.TaxonBackbonePath)
	if err != nil {
		slog.Error("Failed to open backbone taxon file", err)
		return
	}
	defer file.Close()

	var count int = 0
	scanner := bufio.NewScanner(file)
	var tempArray []string

	for scanner.Scan() {
		count++
		var text = scanner.Text()
		fields := strings.Split(text, "\t")

		if fields[11] != "species" {
			continue
		}

		if fields[7] == "" { // Empty species name
			continue
		}

		if fields[17] != "Animalia" && fields[17] != "Plantae" {
			continue
		}

		insertString := fmt.Sprintf("(%s, '%s', '%s', '%s', '%s', '%s', '%s', '%s')", fields[0], safeQuotes(fields[7]), safeQuotes(fields[17]), safeQuotes(fields[18]), safeQuotes(fields[19]), safeQuotes(fields[20]), safeQuotes(fields[21]), safeQuotes(fields[22]))
		tempArray = append(tempArray, insertString)

		if len(tempArray)%5000 == 0 {
			slog.Info("Inserting batch records", "count", len(tempArray))
			insert(&tempArray)
		}
	}

	if len(tempArray) > 0 {
		slog.Info("Inserting last batch records", "count", len(tempArray))
		insert(&tempArray)
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Failed to read backbone taxon file", err)
	}
}

func safeQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func insert(tempArray *[]string) {
	_, err := conn.ExecContext(context.Background(), `
		INSERT OR REPLACE INTO taxa
		(TaxonID, ScientificName, TaxonKingdom, TaxonPhylum, TaxonClass, TaxonOrder, TaxonFamily, TaxonGenus)
		VALUES `+strings.Join(*tempArray, ","))
	if err != nil {
		slog.Error("Database error", err)
	}
	*tempArray = nil
}
