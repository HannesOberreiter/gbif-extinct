/* This table is only a temporary table to store gbif export data if using the import script */
CREATE TABLE IF NOT EXISTS import (
	ObservationID BIGINT PRIMARY KEY,
    TaxonID BIGINT NOT NULL,
	CountryCode VARCHAR NOT NULL,
    ObservationDate VARCHAR NOT NULL,
	ObservationDateOriginal VARCHAR NOT NULL,
	CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);