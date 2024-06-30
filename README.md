# GBIF - Extinct

[![DOI](https://zenodo.org/badge/DOI/10.5281/zenodo.12599947.svg)](https://doi.org/10.5281/zenodo.12599947)

<p align="center">
  <img width="180" height="180" src="/assets/icons/apple-touch-icon-180x180.png">
</p>

## A Tool for Exploring Global Biodiversity Information Facility (GBIF) Data with Enhanced Filtering Capabilities and Potential Applications in Uncovering "Forgotten Taxa"

GBIF-Extinct introduces a tool that facilitates the exploration of the Global Biodiversity Information Facility (GBIF) data with unique filtering functionalities not readily available on the official GBIF website. The homepage enables users to search for the latest observation of specific taxa across different countries. Users can refine their search by applying filters based on taxon name, taxonomic rank, and country.

This user-friendly interface provides several advantages:

- Identification of "Forgotten Taxa": By focusing on the latest observation, the homepage can potentially highlight taxa that have not been recently recorded (potentially "forgotten taxa") within specific regions. This can guide researchers and conservationists towards understudied species and areas, promoting targeted biodiversity research and conservation efforts.
- Enhanced Filtering Capabilities: The homepage's ability to filter by taxonomic rank allows users to explore data at different levels of the taxonomic hierarchy, catering to a broader range of research interests.
- Accessibility and User-Friendliness: The readily accessible homepage interface, with its intuitive filtering options, empowers researchers, students, and citizen scientists with a convenient tool to explore and analyze GBIF data efficiently.

Overall, it offers a valuable contribution to the field of biodiversity research by providing a user-friendly and versatile platform for exploring GBIF data, potentially leading to the identification of "forgotten taxa" and promoting a deeper understanding of global biodiversity patterns.

### Caveats

#### Data Quality

The GBIF data is not perfect and contains errors and biases. The data is only as good as the data providers and the data cleaning process.

- We do not filter for data quality, and the data might contain errors, misidentifications, and outdated records.
- We do not filter for data providers. Some data providers do upload unverified data (one example we found following dataset [gbif.org/dataset/6ac3f774-d9fb-4796-b3e9-92bf6c81c084](https://www.gbif.org/dataset/6ac3f774-d9fb-4796-b3e9-92bf6c81c084)).
- We use "Preserved Specimen" as the event date can be the observation date but this assumption is not always true. The event date is sometimes the collection date of relict fragments. Even more problematic if fossils are marked as "Preserved Specimen" as example *Ursus spelaeus* [gbif.org/occurrence/3415351511](https://www.gbif.org/occurrence/3415351511).

#### Completeness

- We don't do an exhaustive search for all taxa and only use the backbone taxonomy from GBIF. The backbone taxonomy is a consensus taxonomy and might not be up to date with the latest taxonomic changes and we do not update frequently the backbone on our side.
- To reduce query time and load on the gbif API, we take some shortcuts when searching for taxa/countries see function `getCountries` [https://github.com/HannesOberreiter/gbif-extinct/blob/main/pkg/gbif/gbif.go](https://github.com/HannesOberreiter/gbif-extinct/blob/main/pkg/gbif/gbif.go).
- Fetching of new data happens at random with a cron job, therefore the data you see on gbif extinct could be outdated by over a year.

### Usage

Above the table you find a filter form. You can filter by taxon name, taxonomic rank, and country. The taxon name search will return all taxa which contain the search string, eg. "apis" will also return "Caledan**apis** peckorum". The taxonomic rank is a dropdown and will return all taxa which are of the selected rank or higher, the search term itself will match with the start of the string, eg. Family "Ap", will return **Ap**idae, **Ap**iaceae etc. The country code is two letter ISO standard, eg. "AT" for Austria. The synonym checkbox will hide all synonyms from the result.

#### Table Columns

- **Scientific Name**: The scientific name of the taxon. Link redirecting to GBIF taxon page.
- **Country**: The country where the taxon was last observed, as two iso code and a unicode flag.
- **Latest Observation**: The latest observation/occurence of the taxon in the country. The date is formatted as "YYYY-MM-DD". Link redirecting to GBIF occurrence page. The date could differ from GBIF as there are multiple GBIF date formats including ranges, only years etc. For ranges we use the first part and if only part of the date is present we use the first of the year, month or day.
- **~Years**: The years since the last observation. The years are calculated from the current date and the latest observation date.
- **Last Fetched**: The date when the data was last fetched from GBIF. The date is formatted as "YYYY-MM-DD". You can click on the date to force a new fetch of the data.
- **Synonym**: The synonym of the taxon. Link redirecting to GBIF taxon page.
- **Taxa**: The taxonomy of the taxon.

## Reference and Citation

You can download our white paper please see [gbif-extinct-white-paper](/assets/gbif-extinct-white-paper.pdf). If you use GBIF-Extinct in your research, please cite the following:

> Oberreiter, H. und Duenser, A. (2024) „HannesOberreiter/gbif-extinct: v1.3.1“. Zenodo. doi: 10.5281/zenodo.12599948.

```bibtex
@software{oberreiter_2024_12599948,
  author       = {Oberreiter, Hannes and Duenser, Anna},
  title        = {HannesOberreiter/gbif-extinct: v1.3.1},
  month        = jun,
  year         = 2024,
  publisher    = {Zenodo},
  version      = {v1.3.1},
  doi          = {10.5281/zenodo.12599948},
  url          = {https://doi.org/10.5281/zenodo.12599948}
}
```

## Development

The project is open-source and contributions are welcome [github.com/HannesOberreiter/gbif-extinct](https://github.com/HannesOberreiter/gbif-extinct). The project is written in Go and uses Echo as a web framework. HTMX, Tailwind CSS and templ are used to build the frontend. As database we use DuckDB as it can be deployed as binary inside the go application and offers fast self-joining queries.

### Pre-requisites

To get development running you will need [Go](https://golang.org/doc/install) the standalone version of [Tailwind CSS Standalone CLI](https://tailwindcss.com/blog/standalone-cli) and [templ](https://templ.guide/).

### Localhost

For ease of development we use [cosmtrek/air](https://github.com/cosmtrek/air) to automatically reload the server when changes are made. See the [config](.air.toml) file for the configuration.

```bash
air
```

### Taxa Data

To migrate taxa into our database, we use the backbone taxonomy from GBIF, see [hosted-datasets.gbif.org/datasets/backbone/README.html](https://hosted-datasets.gbif.org/datasets/backbone/README.html) for details. To fill the database the `Taxon.tsv` and the `simple.txt` ([github.com/gbif/.../backbone-ddl.sql](https://github.com/gbif/checklistbank/blob/master/checklistbank-mybatis-service/src/main/resources/backbone-ddl.sql)).

Running the mutate script will fill the database with the latest backbone taxonomy from GBIF, set synonyms and delete possible taxa which are synonyms for non species rank taxa.

```bash
go run ./scripts/mutate/mutate.go
```

### Other scripts

The `cron` script does run manually a cron job on a defined TaxonID as a parameter or if no parameter is given it will run a few at random.

```bash
go run ./scripts/cron/cron.go <TaxonID>
```

The `import` script will import occurrence zip files from GBIF into the database. The format must be "simple", when exporting from GBIF. The script will take the path to the zip file as a parameter.

```bash
go run ./scripts/import/import.go <path-to-zip-file>
```

### Testing

To run the tests you will need to set the `SQL_PATH` and `ROOT` environment variables. The `SQL_PATH` is the path to the database file (from the root) and `ROOT` is the path to the root of the project.

```bash
SQL_PATH="memory" ROOT=/gbif-extinct go test -v ./...
```

### Docker

GitHub action is used to generate the web-sever as a docker container [hub.docker.com/r/hannesoberreiter/gbif-extinct](https://hub.docker.com/r/hannesoberreiter/gbif-extinct). See the [Dockerfile](Dockerfile) for details of the build and the [docker-compose.yml](docker-compose.yml) for the deployment.
