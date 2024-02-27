package internal

import (
	"log/slog"

	"github.com/spf13/viper"

	_ "github.com/marcboeker/go-duckdb"
)

var Config *config

type config struct {
	SqlPath           string `mapstructure:"SQL_PATH"`
	TaxonBackbonePath string `mapstructure:"TAXON_BACKBONE_PATH"`
	UserAgentPrefix   string `mapstructure:"USER_AGENT_PREFIX"`
}

func init() {
	loadEnv()
}

// loadEnv loads the environment variables from the .env file or the system environment
// most have sane defaults anyway.
// The order is as follows default < .env < system environment.
func loadEnv() {
	slog.Debug("Loading enviroment variables")
	viper.SetDefault("SQL_PATH", "./db/duck.db")
	viper.SetDefault("TAXON_BACKBONE_PATH", "./Taxon.tsv")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		slog.Error("Error loading .env file", "error", err)
	}

	err = viper.Unmarshal(&Config)
	if err != nil {
		slog.Error("Error parsing .env file", "error", err)
	}
	slog.Info("Loaded environment variables", "config", Config)

}
