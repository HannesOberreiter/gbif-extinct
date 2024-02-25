package internal

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/HannesOberreiter/gbif-extinct/pkg/gbif"
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
func SaveObservation(observation [][]gbif.LatestObservation, conn *sql.Conn, ctx context.Context) {
	slog.Info("Updating observations", "taxa", len(observation))
	const stmt = "INSERT INTO observations (ObservationID, TaxonID, CountryCode, ObservationDate, ObservationDateOriginal) VALUES"
	for _, res := range observation {
		var insertString []string
		slog.Info("Inserting new for taxaId", "observations", len(res), "taxaId", res[0].TaxonID)
		clearOldObservations(conn, ctx, res[0].TaxonID)
		for _, obs := range res {
			insertString = append(insertString, fmt.Sprintf("('%s', '%s', '%s', '%s', '%s')", obs.ObservationID, obs.TaxonID, obs.CountryCode, obs.ObservationDate, obs.ObservationOriginalDate))
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

// UpdateLastFetchStatus updates the last fetch status for a taxon
// this function should be called before updating the observations for a taxon
// The LastFetch column is used to determine if a taxon should be fetched at random by the GetOutdatedObservations function
func UpdateLastFetchStatus(taxonID string) bool {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := DB.Exec("UPDATE taxa SET LastFetch = ? WHERE TaxonID = ?", now, taxonID)
	if err != nil {
		slog.Error("Failed to update last fetch status", "error", err)
		return false
	}
	return true
}

// Helper function to get outdated observations at random
// We only want to fetch a few at a time to not overload the GBIF API
// This function is used by the cron job
func GetOutdatedObservations() []string {
	rows, err := DB.Query(`
		SELECT TaxonID 
		FROM taxa  
		WHERE 6 > date_diff('month', today(), LastFetch) OR LastFetch IS NULL 
		USING SAMPLE 5 ROWS`)
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

type Counts struct {
	TaxaCount        int
	ObservationCount int
}

func GetCounts(payload Payload) Counts {
	var err error
	var taxaCount int
	var observationCount int

	observationQuery := sq.Select("COUNT(*)").From("observations").LeftJoin("taxa ON observations.TaxonID = taxa.TaxonID")
	createFilterQuery(&observationQuery, payload)
	err = observationQuery.RunWith(DB).QueryRow().Scan(&observationCount)
	if err != nil {
		slog.Error("Failed to get observation count", "error", err)
	}

	if payload.COUNTRY != nil && *payload.COUNTRY != "" {
		taxaCount = observationCount // There should be only one taxa per observation per country
	} else {
		taxaQuery := sq.Select("COUNT(*)").From("taxa")
		createFilterQuery(&taxaQuery, payload)
		err = taxaQuery.RunWith(DB).QueryRow().Scan(&taxaCount)
		if err != nil {
			slog.Error("Failed to get taxa count", "error", err)
		}
	}

	counts := Counts{taxaCount, observationCount}
	return counts
}

type TableRow struct {
	TaxonID          string
	ScientificName   sql.NullString
	CountryCode      sql.NullString
	CountryCodeClean string
	CountryFlag      string
	LastFetch        sql.NullTime
	ObservationID    sql.NullString
	ObservationDate  sql.NullTime
	ObservedDiff     string

	TaxonKingdom string
	TaxonPhylum  string
	TaxonClass   string
	TaxonOrder   string
	TaxonFamily  string
	Taxa         string
}

type Payload struct {
	ORDER_BY  *string `query:"order_by"`
	ORDER_DIR *string `query:"order_dir"`
	SEARCH    *string `query:"search"`
	COUNTRY   *string `query:"country"`
	RANK      *string `query:"rank"`
	TAXA      *string `query:"taxa"`
	PAGE      *string `query:"page"`
}

var taxonRankMap = map[string]string{"kingdom": "TaxonKingdom", "phylum": "TaxonPhylum", "class": "TaxonClass", "order": "TaxonOrder", "family": "TaxonFamily"}

func createFilterQuery(query *sq.SelectBuilder, payload Payload) {
	if payload.SEARCH != nil && *payload.SEARCH != "" {
		*query = query.Where(sq.ILike{"ScientificName": "%" + *payload.SEARCH + "%"})
	}
	if payload.COUNTRY != nil && *payload.COUNTRY != "" {
		*query = query.Where(sq.Eq{"CountryCode": strings.ToUpper(*payload.COUNTRY)})
	}

	if payload.RANK != nil && *payload.RANK != "" {
		if payload.TAXA != nil && *payload.TAXA != "" {
			if taxonRankMap[strings.ToLower(*payload.RANK)] != "" {
				slog.Info("Rank", "rank", taxonRankMap[strings.ToLower(*payload.RANK)])
				*query = query.Where(sq.ILike{taxonRankMap[strings.ToLower(*payload.RANK)]: *payload.TAXA + "%"})
			}
		}
	}

}

const PageLimit = uint64(100)

func GetTableData(payload Payload) []TableRow {

	query := sq.Select("taxa.TaxonID", "ScientificName", "CountryCode", "LastFetch", "ObservationID", "ObservationDate",
		"TaxonKingdom", "TaxonPhylum", "TaxonClass", "TaxonOrder", "TaxonFamily").From("taxa").JoinClause("LEFT OUTER JOIN observations ON observations.TaxonID = taxa.TaxonID").Limit(PageLimit)

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

	if payload.PAGE != nil && *payload.PAGE != "" {
		page, err := strconv.ParseInt(*payload.PAGE, 0, 64)
		if err != nil {
			slog.Error("Failed to parse page", "error", err)
		} else {
			offset := PageLimit * (uint64(page) - 1)
			query = query.Offset(offset)
		}
	}

	createFilterQuery(&query, payload)

	rows, err := query.RunWith(DB).Query()

	var result []TableRow
	if err != nil {
		slog.Error("Failed to get outdated observations", "error", err)
		return result
	}
	for rows.Next() {
		var row TableRow
		err = rows.Scan(&row.TaxonID, &row.ScientificName, &row.CountryCode, &row.LastFetch, &row.ObservationID, &row.ObservationDate, &row.TaxonKingdom, &row.TaxonPhylum, &row.TaxonClass, &row.TaxonOrder, &row.TaxonFamily)

		taxonFields := []string{row.TaxonKingdom, row.TaxonPhylum, row.TaxonClass, row.TaxonOrder, row.TaxonFamily}
		row.Taxa = ""

		for i, field := range taxonFields {
			if field != "" {
				row.Taxa += field
			} else {
				row.Taxa += "N/A"
			}
			if i != len(taxonFields)-1 {
				row.Taxa += ", "
			}
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
	return fmt.Sprintf("%.1f", years)
}

func countryCodeToFlag(x string) (country, flag string) {
	if len(x) != 2 {
		return x, ""
	}
	if x[0] < 'A' || x[0] > 'Z' || x[1] < 'A' || x[1] > 'Z' {
		return x, ""
	}
	if x[0] == 'Z' && x[1] == 'Z' {
		return x, "🏴‍☠️"
	}
	return x, " " + string('🇦'+rune(x[0])-'A') + string('🇦'+rune(x[1])-'A')
}
