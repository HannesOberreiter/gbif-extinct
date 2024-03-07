// Purpose: Fetch the latest observations for each taxon from the GBIF API
// save them to the database and return the latest observation for each taxon
package gbif

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	api              = "https://api.gbif.org/v1"
	userAgent        = "gbif-extinct"
	endpoint         = "/occurrence/search"
	limit            = 300
	basisOfRecord    = "basis_of_record=MACHINE_OBSERVATION&basis_of_record=OBSERVATION&basis_of_record=HUMAN_OBSERVATION&basis_of_record=PRESERVED_SPECIMEN"
	occurrenceStatus = "occurrenceStatus=PRESENT"
)

const SampleRows = "5"

// Response is the response from the GBIF API for the occurrence search
type Response struct {
	Offset       int
	Limit        int
	EndOfRecords bool
	Count        int
	Results      []Result
	Facets       []Facet
}

type Result struct {
	Key        int
	DatasetKey string
	EventDate  string
}

type Facet struct {
	Field  string
	Counts []Count
}

type Count struct {
	Name  string
	Count int
}

type LatestObservation struct {
	ObservationID           string
	ObservationOriginalDate string
	ObservationDate         string
	CountryCode             string
	TaxonID                 string
}

type Config struct {
	UserAgentPrefix string
}

// Updates the configuration for the GBIF package
func UpdateConfig(config Config) {
	if config.UserAgentPrefix != "" {
		userAgent = config.UserAgentPrefix + "_" + userAgent
	}
}

// FetchLatest fetches the latest observations of a taxon from the GBIF API
func FetchLatest(taxonID string) []LatestObservation {
	slog.Info("Fetching latest observations from gbif", "taxonID", taxonID)
	years := getYears(taxonID)
	if len(years) == 0 {
		slog.Info("No year data found for taxon")
		return nil
	}
	countries := getCountries(taxonID, years)

	baseUrl := endpoint + "?limit=" + fmt.Sprint(limit) + "&" + basisOfRecord + "&" + occurrenceStatus + "&" + "taxonKey=" + taxonID
	var result []LatestObservation
	for key, year := range countries {
		i := 0
		var observations []LatestObservation
		for {
			offset := i * limit
			i++

			var response Response
			fetchUrl := baseUrl + "&year=" + year + "&country=" + key + "&offset=" + fmt.Sprint(offset)
			body := internalFetch(fetchUrl)
			if body == nil {
				log.Default().Println("Failed to fetch data with Url: " + fetchUrl)
				break
			}
			json.Unmarshal(body, &response)
			breakEarly := false

			if response.Count < 0 {
				slog.Info("No more rows found for given taxa", "taxonID", taxonID)
				break
			} else {
				for _, result := range response.Results {

					if len(result.EventDate) < 4 {
						continue
					}

					cleanDate := cleanDate(result.EventDate)

					if len(observations) > 1 && cleanDate <= observations[len(observations)-1].ObservationDate {
						continue
					}

					observations = append(observations, LatestObservation{
						ObservationID:           fmt.Sprint(result.Key),
						ObservationOriginalDate: result.EventDate,
						ObservationDate:         cleanDate,
						CountryCode:             key,
						TaxonID:                 taxonID,
					})

					// Escape hatch if we already on the last day of the year
					if len(result.EventDate) >= 10 {
						breakYear := result.EventDate[:4]
						breakDay := breakYear + "-12-31"
						if result.EventDate[:10] >= breakDay {
							breakEarly = true
							break
						}
					}

				}
			}

			if response.EndOfRecords || breakEarly {
				break
			}

			time.Sleep(1 * time.Second) // Prevent overload of the GBIF API
		}

		sort.Slice(observations, func(a, b int) bool {
			return observations[b].ObservationDate < observations[a].ObservationDate
		})
		if len(observations) > 0 {
			result = append(result, observations[0])
		}

	}

	return result
}

// SaveObservation saves the latest observation for each taxon
// It first clears the old observations for each taxon before inserting the new ones
// to improve performance each insert contains alls new observations for this taxa at once
func SaveObservation(observation [][]LatestObservation, conn *sql.Conn, ctx context.Context) {
	slog.Info("Updating observations", "taxa", len(observation))
	const stmt = "INSERT INTO observations (ObservationID, TaxonID, CountryCode, ObservationDate, ObservationDateOriginal) VALUES"
	for _, res := range observation {
		var insertString []string
		clearOldObservations(conn, ctx, res[0].TaxonID)
		slog.Info("Inserting new for taxaId", "observations", len(res), "taxaId", res[0].TaxonID)
		for _, obs := range res {
			insertString = append(insertString, fmt.Sprintf("('%s', '%s', '%s', '%s', '%s')", obs.ObservationID, obs.TaxonID, obs.CountryCode, obs.ObservationDate, obs.ObservationOriginalDate))
		}
		query := stmt + strings.Join(insertString, ",") + " ON CONFLICT DO NOTHING;"
		_, err := conn.ExecContext(ctx, query)
		if err != nil {
			slog.Error("Database error on inserting new observations", err)
		}
	}
	err := conn.Close()
	if err != nil {
		slog.Error("Failed to close connection", err)
	}
}

// Get the synonym id for a taxon id, this is used if fetch is called on a synonym
func GetSynonymID(db *sql.DB, taxonID string) (string, error) {
	var synonymID sql.NullString
	err := db.QueryRow("SELECT SynonymID FROM taxa WHERE TaxonID = ?", taxonID).Scan(&synonymID)
	if err != nil {
		slog.Error("Failed to get SynonymID", "error", err)
		return "", errors.New("failed to fetch")
	}
	if !synonymID.Valid {
		return "", errors.New("no id found")
	}
	return synonymID.String, nil
}

// Helper function to get outdated observations at random
// We only want to fetch a few at a time to not overload the GBIF API
// This function is used by the cron job
func GetOutdatedObservations(db *sql.DB) []string {
	rows, err := db.Query(`
		SELECT TaxonID 
		FROM taxa  
		WHERE (6 > date_diff('month', today(), LastFetch) OR LastFetch IS NULL) AND isSynonym = FALSE
		USING SAMPLE` + SampleRows + ` ROWS`)
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

// UpdateLastFetchStatus updates the last fetch status for a taxon
// this function should be called before updating the observations for a taxon
// The LastFetch column is used to determine if a taxon should be fetched at random by the GetOutdatedObservations function
func UpdateLastFetchStatus(db *sql.DB, taxonID string) bool {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec("UPDATE taxa SET LastFetch = ? WHERE SynonymID = ? OR TaxonID = ?", now, taxonID, taxonID)
	if err != nil {
		slog.Error("Failed to update last fetch status", "error", err)
		return false
	}
	return true
}

func internalFetch(url string) []byte {
	client := http.Client{
		Timeout: time.Second * 2,
	}
	req, err := http.NewRequest(http.MethodGet, api+url, nil)
	if err != nil {
		log.Default().Printf("Failed to fetch data: %s", err)
		return nil
	}
	req.Header.Set("User-Agent", userAgent)

	res, getErr := client.Do(req)
	if getErr != nil {
		log.Default().Printf("Failed to fetch data: %s", getErr)
		return nil
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		log.Default().Printf("Failed to read body: %s", readErr)
		return nil
	}

	if res.StatusCode != 200 {
		log.Default().Printf("Failed to fetch data, StatusCode: %s", res.Status)
		return nil
	}

	return body
}

// Helper function to get the years of observations via facet from the API
func getYears(taxonID string) []int {
	var years []int

	year := time.Now().Year() + 1
	url := endpoint + "?facetMultiselect=true&facet=year&facetLimit=5000&taxonKey=" + taxonID + "&year=" + fmt.Sprint(year)

	body := internalFetch(url)
	if body == nil {
		log.Default().Print("Failed to fetch years data")
		return years
	}
	var response Response
	json.Unmarshal(body, &response)
	if len(response.Facets) > 0 {
		for _, facet := range response.Facets {
			if facet.Field == "YEAR" {
				for _, count := range facet.Counts {
					year, err := strconv.Atoi(count.Name)
					if err != nil {
						log.Default().Printf("Failed to convert year to int: %s", err)
						continue
					}
					years = append(years, year)
				}
			}
		}
	}

	sort.Slice(years, func(a, b int) bool {
		return years[b] < years[a]
	})
	return years
}

// Helper function to get the countries of observations via facet from the API
func getCountries(taxonID string, years []int) map[string]string {
	countriesMap := make(map[string]string)

	countries := make(map[string]string)
	i := 0
	for _, year := range years {
		i++
		// We only go 10 deep if we miss any countries so be it
		if i > 10 {
			break
		}

		url := endpoint + "?facet=country&facetLimit=5000&taxonKey=" + taxonID + "&year=" + fmt.Sprint(year)
		body := internalFetch(url)
		var response Response
		json.Unmarshal(body, &response)

		if len(response.Facets) > 0 {
			for _, facet := range response.Facets {
				if facet.Field == "COUNTRY" {
					for _, count := range facet.Counts {
						if count.Name == "" {
							continue
						}
						if _, value := countries[count.Name]; !value {
							countries[count.Name] = count.Name
							countriesMap[count.Name] = fmt.Sprint(year)
						}
					}
				}
			}
		}
	}

	return countriesMap
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

// We are only interested in the latest observation for each taxon, so we clear the old ones before inserting new ones
// runs in the same transaction as SaveObservation
func clearOldObservations(conn *sql.Conn, ctx context.Context, taxonID string) {
	res, err := conn.ExecContext(ctx, "DELETE FROM observations WHERE TaxonID = ?", taxonID)
	if err != nil {
		slog.Error("Database error on clearing old observations", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		slog.Error("Failed to get affected rows", err)
	}
	slog.Info("Deleted old observations", "taxonID", taxonID, "affected", affected)
}
