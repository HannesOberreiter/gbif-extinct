package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"
)

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

const (
	api              = "https://api.gbif.org/v1"
	userAgent        = "gbif-extinct"
	endpoint         = "/occurrence/search"
	limit            = 300
	basisOfRecord    = "basis_of_record=MACHINE_OBSERVATION&basis_of_record=OBSERVATION&basis_of_record=HUMAN_OBSERVATION&basis_of_record=PRESERVED_SPECIMEN"
	occurrenceStatus = "occurrenceStatus=PRESENT"
)

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

type LatestObservation struct {
	ObservationID   string
	ObservationDate string
	CountryCode     string
	TaxonID         string
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
			var response Response
			fetchUrl := baseUrl + "&year=" + year + "&country=" + key + "&offset=" + fmt.Sprint(offset)
			body := internalFetch(fetchUrl)
			if body == nil {
				log.Default().Println("Failed to fetch data with Url: " + fetchUrl)
				break
			}
			json.Unmarshal(body, &response)
			if response.Count < 0 {
				slog.Info("No more rows found for given taxa", "taxonID", taxonID)
				break
			} else {
				for _, result := range response.Results {
					observations = append(observations, LatestObservation{
						ObservationID:   fmt.Sprint(result.Key),
						ObservationDate: result.EventDate,
						CountryCode:     key,
						TaxonID:         taxonID,
					})
				}
			}
			if response.EndOfRecords {
				break
			}
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
