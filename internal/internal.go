package internal

import (
	"database/sql"
	"log"
	"log/slog"

	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
	"github.com/spf13/viper"
)

var (
	DB     *sql.DB
	Config *config
)

type config struct {
	ROOT               string `mapstructure:"ROOT"`
	SqlPath            string `mapstructure:"SQL_PATH"`
	TaxonBackbonePath  string `mapstructure:"TAXON_BACKBONE_PATH"`
	TaxonSimplePath    string `mapstructure:"TAXON_SIMPLE_PATH"`
	UserAgentPrefix    string `mapstructure:"USER_AGENT_PREFIX"`
	CronJobIntervalSec int    `mapstructure:"CRON_JOB_INTERVAL_SEC"`
}

func Load() {
	slog.Debug("Initializing internal package")
	loadEnv()
	loadDb()
}

// Initialize the database connection
// DuckDB only supports one connection at a time,
// therefore as long as the server is running we cannot connect to the database externally
func loadDb() {
	var err error
	var dbPath string
	if Config.SqlPath == "memory" {
		slog.Info("No database path set, using in-memory database")
		dbPath = ""
	} else {
		dbPath = Config.ROOT + Config.SqlPath
	}

	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		slog.Debug("Failed to connect to database.", "path", dbPath)
		log.Fatal(err)
	}

	DB.Exec("PRAGMA foreign_keys = ON;")
	DB.Exec("pragma journal_mode = WAL;")
	DB.Exec("pragma synchronous = normal;")
	DB.Exec("pragma journal_size_limit = 6144000;")

	slog.Info("Connected to database.", "path", dbPath)
}

// loadEnv loads the environment variables from the .env file or the system environment
// most have sane defaults anyway.
// The order is as follows default < .env < system environment.
func loadEnv() {
	slog.Debug("Loading environment variables")

	viper.SetDefault("SQL_PATH", "/db/lite.db")
	viper.SetDefault("TAXON_BACKBONE_PATH", "/Taxon.tsv")
	viper.SetDefault("TAXON_SIMPLE_PATH", "/simple.txt")
	viper.SetDefault("USER_AGENT_PREFIX", "local")
	viper.SetDefault("CRON_JOB_INTERVAL_SEC", 0)
	viper.SetDefault("ROOT", ".")

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		slog.Warn("Error loading .env file", "error", err)
	}

	err = viper.Unmarshal(&Config)
	if err != nil {
		slog.Error("Error parsing .env file", "error", err)
	}
	slog.Info("Loaded environment variables", "config", Config)
}
