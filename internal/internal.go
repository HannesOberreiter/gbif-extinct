package internal

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/HannesOberreiter/gbif-extinct/pkg"
	sq "github.com/Masterminds/squirrel"
	_ "github.com/marcboeker/go-duckdb"
)

var DB *sql.DB

func init() {
	slog.Debug("Initializing internal package")
	loadDb()
	Migrations()
}

// Initialize the database connection
// DuckDB only supports one connection at a time,
// therefore as long as the server is running we cannot connect to the database externally
func loadDb() {
	var err error
	DB, err = sql.Open("duckdb", Config.SqlPath)
	if err != nil {
		slog.Debug("Failed to connect to database.", "path", Config.SqlPath)
		log.Fatal(err)
	}
	slog.Info("Connected to database.", "path", Config.SqlPath)
}

// SaveObservation saves the latest observation for each taxon
// It first clears the old observations for each taxon before inserting the new ones
// to improve performance each insert contains alls new observations for this taxa at once
func SaveObservation(observation [][]pkg.LatestObservation, conn *sql.Conn, ctx context.Context) {
	slog.Info("Updating observations", "taxa", len(observation))
	const stmt = "INSERT INTO observations (ObservationID, TaxonID, CountryCode, ObservationDate, ObservationDateOriginal) VALUES"
	for _, res := range observation {
		var insertString []string
		slog.Info("Inserting new for taxaId", "observations", len(res), "taxaId", res[0].TaxonID)
		clearOldObservations(conn, ctx, res[0].TaxonID)
		for _, obs := range res {
			insertString = append(insertString, fmt.Sprintf("('%s', '%s', '%s', '%s', '%s')", obs.ObservationID, obs.TaxonID, obs.CountryCode, cleanDate(obs.ObservationDate), obs.ObservationDate))
		}
		query := stmt + strings.Join(insertString, ",") + " ON CONFLICT DO NOTHING;"
		_, err := conn.ExecContext(ctx, query)
		if err != nil {
			slog.Error("Database error on inserting new observations", err)
		}
	}
}

// We are only interested in the latest observation for each taxon, so we clear the old ones before inserting new ones
// runs in the same transaction as SaveObservation
func clearOldObservations(conn *sql.Conn, ctx context.Context, taxonID string) {
	slog.Info("Clearing old observations")
	_, err := conn.ExecContext(ctx, "DELETE FROM observations WHERE TaxonID = ?", taxonID)
	if err != nil {
		slog.Error("Database error on clearing old observations", err)
	}
}

// Clean observation date to be in the format of YYYY-MM-DD
func cleanDate(date string) string {
	if date == "" {
		return ""
	}

	var dateParts []string
	if strings.Contains(date, "/") {
		dateParts = strings.Split(date, "/")
	} else {
		dateParts = []string{date}
	}

	// Remove time if it exists
	dateParts = strings.Split(dateParts[0], " ")
	dateParts = strings.Split(dateParts[0], "T")

	dateParts = strings.Split(dateParts[0], "-")
	if len(dateParts) == 1 {
		return dateParts[0] + "-01-01"
	} else if len(dateParts) == 2 {
		return dateParts[0] + "-" + dateParts[1] + "-01"
	} else {
		return dateParts[0] + "-" + dateParts[1] + "-" + dateParts[2]
	}
}

// UpdateLastFetchStatus updates the last fetch status for a taxon
// this function should be called before updating the observations for a taxon
// The LastFetch column is used to determine if a taxon should be fetched at random by the GetOutdatedObservations function
func UpdateLastFetchStatus(taxonID string) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := DB.Exec("UPDATE taxa SET LastFetch = ? WHERE TaxonID = ?", now, taxonID)
	if err != nil {
		log.Fatal(err)
	}
}

// Helper function to get outdated observations at random
// We only want to fetch a few at a time to not overload the GBIF API
// This function is used by the cron job
func GetOutdatedObservations() []string {
	rows, err := DB.Query(`
		SELECT TaxonID 
		FROM taxa  
		WHERE 6 > date_diff('month', today(), LastFetch) OR LastFetch IS NULL 
		USING SAMPLE 10 ROWS`)
	var taxonIDs []string
	if err != nil {
		slog.Error("Failed to get outdated observations", "error", err)
		return taxonIDs
	}
	for rows.Next() {
		var taxonID string
		err = rows.Scan(&taxonID)
		if err != nil {
			log.Fatal(err)
		}
		taxonIDs = append(taxonIDs, taxonID)
	}

	return taxonIDs
}

type TableRow struct {
	TaxonID          string
	ScientificName   sql.NullString
	CountryCode      sql.NullString
	CountryCodeClean string
	CountryFlag      string
	LastFetch        sql.NullTime
	ObservationID    string
	ObservationDate  sql.NullTime
	ObservedDiff     string

	TaxonKingdom sql.NullString
	TaxonPhylum  sql.NullString
	TaxonClass   sql.NullString
	TaxonOrder   sql.NullString
	TaxonFamily  sql.NullString
	Taxa         string
}

type Payload struct {
	ORDER_BY  *string `query:"order_by"`
	ORDER_DIR *string `query:"order_dir"`
}

func GetTableData(payload Payload) []TableRow {

	slog.Info("Payload", "payload", payload)

	query := sq.Select("taxa.TaxonID", "ScientificName", "CountryCode", "LastFetch", "ObservationID", "ObservationDate",
		"TaxonKingdom", "TaxonPhylum", "TaxonClass", "TaxonOrder", "TaxonFamily").From("taxa").Join("observations ON observations.TaxonID = taxa.TaxonID").Limit(100)

	var direction string
	if payload.ORDER_DIR == nil || *payload.ORDER_DIR == "asc" {
		direction = "ASC NULLS LAST"
	} else {
		direction = "DESC  NULLS LAST"
	}

	if payload.ORDER_BY == nil || *payload.ORDER_BY == "" || *payload.ORDER_BY == "date" {
		query = query.OrderBy("ObservationDate " + direction)
	} else if *payload.ORDER_BY == "name" {
		query = query.OrderBy("ScientificName " + direction)
	} else if *payload.ORDER_BY == "fetch" {
		query = query.OrderBy("LastFetch " + direction)
	}

	rows, err := query.RunWith(DB).Query()

	var result []TableRow
	if err != nil {
		slog.Error("Failed to get outdated observations", "error", err)
		return result
	}
	for rows.Next() {
		var row TableRow
		err = rows.Scan(&row.TaxonID, &row.ScientificName, &row.CountryCode, &row.LastFetch, &row.ObservationID, &row.ObservationDate, &row.TaxonKingdom, &row.TaxonPhylum, &row.TaxonClass, &row.TaxonOrder, &row.TaxonFamily)

		row.Taxa = ""
		if row.TaxonKingdom.Valid {
			row.Taxa += row.TaxonKingdom.String
		}
		if row.TaxonPhylum.Valid {
			row.Taxa += ", " + row.TaxonPhylum.String
		}
		if row.TaxonClass.Valid {
			row.Taxa += ", " + row.TaxonClass.String
		}
		if row.TaxonOrder.Valid {
			row.Taxa += ", " + row.TaxonOrder.String
		}
		if row.TaxonFamily.Valid {
			row.Taxa += ", " + row.TaxonFamily.String
		}

		if row.ObservationDate.Valid {
			row.ObservedDiff = calculateTimeSinceYears(row.ObservationDate.Time)
		} else {
			row.ObservedDiff = "N/A"
		}

		if row.CountryCode.Valid {
			row.CountryCodeClean, row.CountryFlag = countryCodeToFlag(row.CountryCode.String)
		}

		if err != nil {
			slog.Error("Failed to get outdated observations", "error", err)
		}

		result = append(result, row)
	}

	return result
}

func calculateTimeSinceYears(t time.Time) string {
	years := time.Since(t).Hours() / 24 / 365
	return fmt.Sprintf("~%.2f", years)
}

func countryCodeToFlag(x string) (country, flag string) {
	if len(x) != 2 {
		return x, ""
	}
	if x[0] < 'A' || x[0] > 'Z' || x[1] < 'A' || x[1] > 'Z' {
		return x, ""
	}
	return x, " " + string('🇦'+rune(x[0])-'A') + string('🇦'+rune(x[1])-'A')
}
