// Manually upload observations from gbif simple file download.
package main

import (
	"archive/zip"
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/HannesOberreiter/gbif-extinct/internal"
	"github.com/HannesOberreiter/gbif-extinct/pkg/gbif"
)

var conn *sql.Conn

func main() {
	slog.Info("Starting loading observations")

	if len(os.Args) < 2 {
		slog.Error("No file path given")
		return
	}

	filePath := os.Args[1]
	if filePath == "" {
		slog.Error("No file path given")
		return
	}

	internal.Load()
	internal.Migrations(internal.DB, internal.Config.ROOT)

	var err error
	conn, err = internal.DB.Conn(context.Background())
	if err != nil {
		slog.Error("Failed to connect to database", err)
		return
	}
	defer conn.Close()

	clearImport()
	importZIP(filePath)
	removeOldObservations()
	moveToObservations()
	updateLastFetchStatus()
	clearImport()

	conn.Close()
}

// Clear import table after and before import
func clearImport() {
	_, err := conn.ExecContext(context.Background(), "DELETE FROM import")
	if err != nil {
		slog.Error("Failed to clear import table", err)
		log.Fatal(err)
	}
}

// Import gbif "simple" export zip into import table
func importZIP(filePath string) {
	slog.Info("Importing zip file", "filePath", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		slog.Error("Failed to gbif zip file", err)
		log.Fatal(err)
	}
	defer file.Close()

	// read zip file
	fileInfo, err := file.Stat()
	if err != nil {
		slog.Error("Failed to get file info", err)
		log.Fatal(err)
	}

	reader, err := zip.NewReader(file, fileInfo.Size())
	if err != nil {
		slog.Error("Failed to read zip file", err)
		log.Fatal(err)
	}

	for _, zf := range reader.File {
		zfFile, err := zf.Open()
		if err != nil {
			slog.Error("Failed to open file in zip", err)
			continue
		}
		defer zfFile.Close()

		scanner := bufio.NewScanner(zfFile)
		var tempArray []string

		var count int = 0
		for scanner.Scan() {
			var text = scanner.Text()
			fields := strings.Split(text, "\t")

			/* gbifID  datasetKey      occurrenceID    kingdom phylum  class   order   family  genus   species infraspecificEpithet    taxonRank       scientificName  verbatimScientificName    verbatimScientificNameAuthorship        countryCode     locality        stateProvince   occurrenceStatus        individualCount publishingOrgKey decimalLatitude  decimalLongitude        coordinateUncertaintyInMeters   coordinatePrecision     elevation       elevationAccuracy       depth   depthAccuracy   eventDate day     month   year    taxonKey        speciesKey      basisOfRecord   institutionCode collectionCode  catalogNumber   recordNumber    identifiedBy    dateIdentified    license rightsHolder    recordedBy      typeStatus      establishmentMeans      lastInterpreted mediaType       issue */

			data := struct {
				TaxonID                 string
				ObservationID           string
				taxonRank               string
				CountryCode             string
				ObservationDateOriginal string
			}{
				TaxonID:                 fields[34],
				ObservationID:           fields[0],
				taxonRank:               fields[11],
				CountryCode:             fields[15],
				ObservationDateOriginal: fields[29],
			}

			if data.taxonRank != "SPECIES" {
				continue
			}

			if data.CountryCode == "" || data.CountryCode == " " || data.CountryCode == "\\N" {
				continue
			}

			if data.ObservationDateOriginal == "" || data.ObservationDateOriginal == " " || data.ObservationDateOriginal == "\\N" {
				continue
			}

			cleanDate := gbif.CleanDate(data.ObservationDateOriginal)

			insertString := fmt.Sprintf("('%s', '%s', '%s', '%s', '%s')", safeQuotes(data.ObservationID), safeQuotes(data.TaxonID), safeQuotes(data.CountryCode), safeQuotes(data.ObservationDateOriginal), safeQuotes(cleanDate))

			tempArray = append(tempArray, insertString)
			count++

			if len(tempArray)%20000 == 0 {
				slog.Info("Inserting batch records", "count", len(tempArray))
				insert(&tempArray, "import")
			}
		}

		if err := scanner.Err(); err != nil {
			slog.Error("Failed to read backbone taxon file", err)
		}

		if len(tempArray) > 0 {
			slog.Info("Inserting last batch records", "count", len(tempArray))
			insert(&tempArray, "import")
		}
	}

}

func updateLastFetchStatus() {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := conn.ExecContext(context.Background(), "UPDATE taxa SET LastFetch = ? WHERE SynonymID IN (SELECT DISTINCT(TaxonID) FROM import) OR TaxonID IN (SELECT DISTINCT(TaxonID) FROM import)", now)
	if err != nil {
		slog.Error("Failed to update last fetch status", "error", err)
		log.Fatal(err)
	}
}

// Before moving imported data to observation table, remove old observations
func removeOldObservations() {
	_, err := conn.ExecContext(context.Background(), "DELETE FROM observations WHERE TaxonID IN (SELECT DISTINCT(TaxonID) FROM import)")
	if err != nil {
		slog.Error("Failed to clear observations table", err)
		log.Fatal(err)
	}
}

// Move imported data to observation table
func moveToObservations() {
	stmt := `WITH tmp AS (
				SELECT
				TaxonID,
				ObservationID,
				CountryCode,
				ObservationDateOriginal,
				ObservationDate,
    			row_number() OVER (PARTITION BY TaxonID, CountryCode ORDER  BY ObservationDate DESC) AS Row
				FROM import 
				ORDER BY ObservationDate
			) SELECT ObservationID, TaxonID, CountryCode, ObservationDateOriginal, ObservationDate FROM tmp WHERE Row = 1;`
	res, err := conn.QueryContext(context.Background(), stmt)
	if err != nil {
		slog.Error("Failed to get window of import table", err)
		log.Fatal(err)
	}
	defer res.Close()

	var tempArray []string
	var count int = 0
	for res.Next() {
		var data struct {
			ObservationID           string
			TaxonID                 string
			CountryCode             string
			ObservationDateOriginal string
			ObservationDate         string
		}
		err := res.Scan(&data.ObservationID, &data.TaxonID, &data.CountryCode, &data.ObservationDateOriginal, &data.ObservationDate)
		if err != nil {
			slog.Error("Failed to scan data", err)
			log.Fatal(err)
		}

		insertString := fmt.Sprintf("('%s', '%s', '%s', '%s', '%s')", safeQuotes(data.ObservationID), safeQuotes(data.TaxonID), safeQuotes(data.CountryCode), safeQuotes(data.ObservationDateOriginal), safeQuotes(data.ObservationDate))
		tempArray = append(tempArray, insertString)
		count++

		if len(tempArray)%20000 == 0 {
			slog.Info("Move batch records", "count", len(tempArray))
			insert(&tempArray, "observations")
		}
	}

	if len(tempArray) > 0 {
		slog.Info("Moving last batch records", "count", len(tempArray))
		insert(&tempArray, "observations")
	}

}

func safeQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func insert(tempArray *[]string, table string) {
	_, err := conn.ExecContext(context.Background(), `
		INSERT OR REPLACE INTO `+table+`
		(ObservationID, TaxonID, CountryCode, ObservationDateOriginal, ObservationDate)
		VALUES `+strings.Join(*tempArray, ","))
	if err != nil {
		slog.Error("Database error", err)
		log.Fatal(err)
	}
	*tempArray = nil
}
