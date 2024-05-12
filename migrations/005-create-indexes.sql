/* Indexes */
CREATE INDEX idx_taxa_ScientificName ON taxa(ScientificName);
CREATE INDEX idx_taxa_LastFetch ON taxa(LastFetch);
CREATE INDEX idx_taxa_isSynonym ON taxa(isSynonym);
CREATE INDEX idx_taxa_SynonymID ON taxa(SynonymID);


CREATE INDEX idx_observations_TaxonID ON observations(TaxonID);
CREATE INDEX idx_observations_CountryCode ON observations(CountryCode);
CREATE INDEX idx_observations_ObservationDate ON observations(ObservationDate);
