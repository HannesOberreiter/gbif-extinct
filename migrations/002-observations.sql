/* Observations of taxa 1:n with taxa table in TaxonID as foreign key */
CREATE TABLE IF NOT EXISTS observations (
	ObservationID BIGINT PRIMARY KEY,
	CountryCode VARCHAR NOT NULL,
	ObservationDate DATE NOT NULL,
	ObservationDateOriginal VARCHAR NOT NULL,
	CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	TaxonID BIGINT NOT NULL
);