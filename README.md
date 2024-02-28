# GBIF - Extinct

Project to keep track for the last seen observations in each country for species taxa on GBIF. This project was made to help working with the GBIF data, as you cannot easily filter for latest observation of multiple taxa on GBIF.

## Taxa

To migrate taxa into our database, we use the backbone taxonomy from GBIF, see [hosted-datasets.gbif.org/datasets/backbone/README.html](https://hosted-datasets.gbif.org/datasets/backbone/README.html) for details. To fill the database the `Taxon.tsv` and the `simple.txt` ([github.com/gbif/.../backbone-ddl.sql](https://github.com/gbif/checklistbank/blob/master/checklistbank-mybatis-service/src/main/resources/backbone-ddl.sql)).

Running the mutate script will fill the database with the latest backbone taxonomy from GBIF, set synonyms and delete possible taxa which are synonyms for non species rank taxa.

```bash
go run ./scripts/mutate/mutate.go
```

## Development

For ease of development we use [cosmtrek/air](https://github.com/cosmtrek/air) to automatically reload the server when changes are made. See the [config](.air.toml) file for the configuration.

### Pre-requisites

To get development running you will need [Go](https://golang.org/doc/install) the standalone version of [Tailwind CSS Standalone CLI](https://tailwindcss.com/blog/standalone-cli) and [templ](https://templ.guide/).

## CI/CD

For ease of deployment, a GitHub action is used to generate the web-sever as a docker container [hub.docker.com/r/hannesoberreiter/gbif-extinct](https://hub.docker.com/r/hannesoberreiter/gbif-extinct).
