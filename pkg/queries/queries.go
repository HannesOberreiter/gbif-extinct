package queries

import (
	"database/sql"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type Query struct {
	ORDER_BY      string
	ORDER_DIR     string
	SEARCH        string
	COUNTRY       string
	RANK          string
	TAXA          string
	PAGE          string
	SHOW_SYNONYMS bool
}

type Counts struct {
	TaxaCount        int
	ObservationCount int
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

	IsSynonym   bool
	SynonymName sql.NullString
	SynonymID   sql.NullString

	TaxonKingdom string
	TaxonPhylum  string
	TaxonClass   string
	TaxonOrder   string
	TaxonFamily  string
	Taxa         string
}

var _taxonRankMap = map[string]string{"kingdom": "TaxonKingdom", "phylum": "TaxonPhylum", "class": "TaxonClass", "order": "TaxonOrder", "family": "TaxonFamily"}

var _selectArray = []string{"taxa.TaxonID", "ScientificName", "CountryCode", "LastFetch", "ObservationID", "ObservationDate", "TaxonKingdom", "TaxonPhylum", "TaxonClass", "TaxonOrder", "TaxonFamily", "isSynonym", "SynonymName", "SynonymID"}

const DefaultPageLimit = uint64(100)
const IncreasedPageLimit = uint64(10_000)

// Create a new query object with default values or set values from payload struct
func NewQuery(payload any) Query {
	q := Query{
		ORDER_BY:      "date",
		ORDER_DIR:     "asc",
		SEARCH:        "",
		COUNTRY:       "",
		RANK:          "",
		TAXA:          "",
		PAGE:          "1",
		SHOW_SYNONYMS: false,
	}

	if payload != nil {
		typeOfPayload := reflect.TypeOf(payload)
		if typeOfPayload.Kind() == reflect.Struct {
			payloadValue := reflect.ValueOf(payload)
			for i := 0; i < typeOfPayload.NumField(); i++ {
				field := typeOfPayload.Field(i)
				fieldValue := payloadValue.Field(i)

				val, ok := getFieldValue(fieldValue)
				if !ok || val == nil {
					continue
				}
				switch field.Name {
				case "ORDER_BY":
					q.ORDER_BY = val.(string)
				case "ORDER_DIR":
					q.ORDER_DIR = val.(string)
				case "SEARCH":
					q.SEARCH = val.(string)
				case "COUNTRY":
					q.COUNTRY = val.(string)
				case "RANK":
					q.RANK = val.(string)
				case "TAXA":
					q.TAXA = val.(string)
				case "PAGE":
					q.PAGE = val.(string)
				case "SHOW_SYNONYMS":
					if reflect.TypeOf(val).Kind() == reflect.Bool {
						q.SHOW_SYNONYMS = val.(bool)
					}
				}
			}
		}

	}

	return q
}

// Get the counts of taxa and observations based on the query
func GetCounts(db *sql.DB, q Query) Counts {
	var err error
	var taxaCount int
	var observationCount int

	observationQuery := sq.Select("COUNT(DISTINCT(observations.TaxonID, observations.CountryCode))").From("observations").LeftJoin("taxa ON observations.TaxonID = taxa.SynonymID")
	createFilterQuery(&observationQuery, q)
	err = observationQuery.RunWith(db).QueryRow().Scan(&observationCount)
	if err != nil {
		slog.Error("Failed to get observation count", "error", err)
	}

	if q.COUNTRY != "" {
		taxaCount = observationCount // There should be only one taxa per observation per country
	} else {
		taxaQuery := sq.Select("COUNT(DISTINCT taxa.SynonymID)").From("taxa")

		createFilterQuery(&taxaQuery, q)
		err = taxaQuery.RunWith(db).QueryRow().Scan(&taxaCount)
		if err != nil {
			slog.Error("Failed to get taxa count", "error", err)
		}
	}

	counts := Counts{taxaCount, observationCount}
	return counts
}

// Get the table data based on the query
func GetTableData(db *sql.DB, q Query, increaseLimit ...bool) []TableRow {
	query := sq.Select(_selectArray...).From("taxa").JoinClause("LEFT OUTER JOIN observations ON observations.TaxonID = taxa.SynonymID").Limit(DefaultPageLimit)

	if increaseLimit != nil && increaseLimit[0] {
		query = query.Limit(IncreasedPageLimit)
	}

	var direction string
	if q.ORDER_DIR == "asc" {
		direction = "ASC NULLS LAST"
	} else {
		direction = "DESC  NULLS LAST"
	}

	if q.ORDER_BY == "" || q.ORDER_BY == "date" {
		query = query.OrderBy("ObservationDate " + direction)
	} else if q.ORDER_BY == "name" {
		query = query.OrderBy("ScientificName " + direction)
	} else if q.ORDER_BY == "fetch" {
		query = query.OrderBy("LastFetch " + direction)
	}

	if q.PAGE != "" {
		page, err := strconv.ParseInt(q.PAGE, 0, 64)
		if err != nil {
			slog.Error("Failed to parse page", "error", err)
		} else {
			offset := DefaultPageLimit * (uint64(page) - 1)
			query = query.Offset(offset)
		}
	}

	createFilterQuery(&query, q)

	rows, err := query.RunWith(db).Query()

	var result []TableRow
	if err != nil {
		slog.Error("Failed to get outdated observations", "error", err)
		return result
	}
	for rows.Next() {
		var row TableRow
		err = rows.Scan(&row.TaxonID, &row.ScientificName, &row.CountryCode, &row.LastFetch, &row.ObservationID, &row.ObservationDate, &row.TaxonKingdom, &row.TaxonPhylum, &row.TaxonClass, &row.TaxonOrder, &row.TaxonFamily, &row.IsSynonym, &row.SynonymName, &row.SynonymID)

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

// Create a CSV string from the table data, this is used for exporting data
func CreateCSV(rows []TableRow) string {
	var csv string
	csv += strings.Join(_selectArray, ",") + "\n"
	for _, row := range rows {
		scientificName := ""
		if row.ScientificName.Valid {
			scientificName = row.ScientificName.String
		}
		countryCode := ""
		if row.CountryCode.Valid {
			countryCode = row.CountryCode.String
		}
		observationID := ""
		if row.ObservationID.Valid {
			observationID = row.ObservationID.String
		}
		observationDate := ""
		if row.ObservationDate.Valid {
			observationDate = row.ObservationDate.Time.Format("2006-01-02")
		}
		synonymName := ""
		if row.SynonymName.Valid {
			synonymName = row.SynonymName.String
		}
		synonymID := ""
		if row.SynonymID.Valid {
			synonymID = row.SynonymID.String
		}
		/* Needs to be same order as _selectArray */
		csv += fmt.Sprintf(
			"%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%t,%s,%s\n", row.TaxonID, scientificName, countryCode, row.LastFetch.Time.Format("2006-01-02"), observationID, observationDate, row.TaxonKingdom, row.TaxonPhylum, row.TaxonClass, row.TaxonOrder, row.TaxonFamily, row.IsSynonym, synonymName, synonymID)
	}
	return csv
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

func createFilterQuery(query *sq.SelectBuilder, q Query) {
	if !q.SHOW_SYNONYMS {
		*query = query.Where(sq.Or{sq.Eq{"isSynonym": false}, sq.Eq{"SynonymID": nil}})
	}

	if q.SEARCH != "" {
		*query = query.Where(sq.ILike{"ScientificName": "%" + q.SEARCH + "%"})
	}
	if q.COUNTRY != "" {
		*query = query.Where(sq.Eq{"CountryCode": strings.ToUpper(q.COUNTRY)})
	}

	if q.RANK != "" {
		if q.TAXA != "" {
			if _taxonRankMap[strings.ToLower(q.RANK)] != "" {
				slog.Info("Rank", "rank", _taxonRankMap[strings.ToLower(q.RANK)])
				*query = query.Where(sq.ILike{_taxonRankMap[strings.ToLower(q.RANK)]: q.TAXA + "%"})
			}
		}
	}
}

// Get the value of a field, handling pointers
func getFieldValue(fieldValue reflect.Value) (interface{}, bool) {
	if fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			return nil, false // Handle nil pointer case
		}
		return fieldValue.Elem().Interface(), true
	}
	return fieldValue.Interface(), true
}

func GetCountTotalTaxa(db *sql.DB) int {
	var count int
	err := sq.Select("COUNT(TaxonID)").From("taxa").Where(sq.Eq{"isSynonym": false}).RunWith(db).QueryRow().Scan(&count)
	if err != nil {
		slog.Error("Failed to get taxa count", "error", err)
	}
	return count
}

func GetCountFetchedLastTwelveMonths(db *sql.DB) int {
	var count int
	err := sq.Select("COUNT(TaxonID)").From("taxa").Where("LastFetch > CURRENT_DATE - INTERVAL 12 MONTH").Where(sq.Eq{"isSynonym": false}).RunWith(db).QueryRow().Scan(&count)
	if err != nil {
		slog.Error("Failed to get taxa count", "error", err)
	}
	return count
}
