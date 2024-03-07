# GBIF - Extinct

## A Tool for Exploring Global Biodiversity Information Facility (GBIF) Data with Enhanced Filtering Capabilities and Potential Applications in Uncovering "Forgotten Taxa"

GBIF-Extinct introduces a tool that facilitates the exploration of the Global Biodiversity Information Facility (GBIF) data with unique filtering functionalities not readily available on the official GBIF website. The homepage enables users to search for the latest observation of specific taxa across different countries, offering insights into spatiotemporal distribution patterns. Users can refine their search by applying filters based on taxon name, taxonomic rank, and country.

This user-friendly interface provides several advantages:

- Identification of "Forgotten Taxa": By focusing on the latest observation, the homepage can potentially highlight taxa that have not been recently recorded (potentially "forgotten taxa") within specific regions. This can guide researchers and conservationists towards understudied species and areas, promoting targeted biodiversity research and conservation efforts.
- Enhanced Filtering Capabilities: The homepage's ability to filter by taxonomic rank allows users to explore data at different levels of the taxonomic hierarchy, catering to a broader range of research interests.
- Accessibility and User-Friendliness: The readily accessible homepage interface, with its intuitive filtering options, empowers researchers, students, and citizen scientists with a convenient tool to explore and analyze GBIF data efficiently.

Overall, it offers a valuable contribution to the field of biodiversity research by providing a user-friendly and versatile platform for exploring GBIF data, potentially leading to the identification of "forgotten taxa" and promoting a deeper understanding of global biodiversity patterns.

## Development

The project is open-source and contributions are welcome [github.com/HannesOberreiter/gbif-extinct](https://github.com/HannesOberreiter/gbif-extinct). The project is written in Go and uses Echo as a web framework. HTMX and Tailwind CSS and templ are used for the frontend. The database is DuckDB as it can be deployed as binary inside the go application.

### Pre-requisites

To get development running you will need [Go](https://golang.org/doc/install) the standalone version of [Tailwind CSS Standalone CLI](https://tailwindcss.com/blog/standalone-cli) and [templ](https://templ.guide/).

### Localhost

For ease of development we use [cosmtrek/air](https://github.com/cosmtrek/air) to automatically reload the server when changes are made. See the [config](.air.toml) file for the configuration.

### Taxa Data

To migrate taxa into our database, we use the backbone taxonomy from GBIF, see [hosted-datasets.gbif.org/datasets/backbone/README.html](https://hosted-datasets.gbif.org/datasets/backbone/README.html) for details. To fill the database the `Taxon.tsv` and the `simple.txt` ([github.com/gbif/.../backbone-ddl.sql](https://github.com/gbif/checklistbank/blob/master/checklistbank-mybatis-service/src/main/resources/backbone-ddl.sql)).

Running the mutate script will fill the database with the latest backbone taxonomy from GBIF, set synonyms and delete possible taxa which are synonyms for non species rank taxa.

```bash
go run ./scripts/mutate/mutate.go
```

### Docker

GitHub action is used to generate the web-sever as a docker container [hub.docker.com/r/hannesoberreiter/gbif-extinct](https://hub.docker.com/r/hannesoberreiter/gbif-extinct).
