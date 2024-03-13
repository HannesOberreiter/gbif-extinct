/* Taxa backbone table */
CREATE TABLE IF NOT EXISTS taxa (
    TaxonID BIGINT PRIMARY KEY,
	ScientificName VARCHAR NOT NULL,
	TaxonKingdom VARCHAR,
	TaxonPhylum VARCHAR,
	TaxonClass VARCHAR,
	TaxonOrder VARCHAR,
	TaxonFamily VARCHAR,
	TaxonGenus VARCHAR,
	LastFetch TIMESTAMP DEFAULT NULL,
	CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
