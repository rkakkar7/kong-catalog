package config

import (
	"fmt"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
	yaml "sigs.k8s.io/yaml/goyaml.v2"
)

type AppConfig struct {
	Addr        string `yaml:"addr"`
	DatabaseURL string `yaml:"database_url"`
	MaxPageSize int    `yaml:"max_page_size"`

	// Database connection configuration
	DBMaxConnections    int           `yaml:"db_max_connections" envconfig:"DB_MAX_CONNECTIONS"`
	DBMinConnections    int           `yaml:"db_min_connections" envconfig:"DB_MIN_CONNECTIONS"`
	DBMaxConnLifetime   time.Duration `yaml:"db_max_conn_lifetime" envconfig:"DB_MAX_CONN_LIFETIME"`
	DBMaxConnIdleTime   time.Duration `yaml:"db_max_conn_idle_time" envconfig:"DB_MAX_CONN_IDLE_TIME"`
	DBConnectTimeout    time.Duration `yaml:"db_connect_timeout" envconfig:"DB_CONNECT_TIMEOUT"`
	DBHealthCheckPeriod time.Duration `yaml:"db_health_check_period" envconfig:"DB_HEALTH_CHECK_PERIOD"`

	// API configuration
	ValidAPIKeys []string `yaml:"valid_api_keys" envconfig:"VALID_API_KEYS"`
}

// global app config
var appConfig *AppConfig

// GetAppConfig gets the app configuration
func GetAppConfig() *AppConfig {
	if appConfig == nil {
		appConfig = &AppConfig{}
	}
	return appConfig
}

// ParseAndLoadConfig parses config from a file
func ParseAndLoadConfig(filename string) error {
	log.Info().Str("filename", filename).Msg("Starting to load config")

	// Ensure appConfig is initialized
	if appConfig == nil {
		appConfig = &AppConfig{}
	}

	// Read config from file first
	configData, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("encountered a problem reading file (%s): %w", filename, err)
	}
	log.Info().Msg("Config file read successfully")

	// Parse into a AppConfig object
	err = yaml.Unmarshal(configData, appConfig)
	if err != nil {
		return err
	}
	log.Info().Str("database_url", appConfig.DatabaseURL).Msg("YAML parsed")

	// Then process environment variables (these will override YAML values)
	err = envconfig.Process("", appConfig)
	if err != nil {
		return fmt.Errorf("encountered a problem reading environment, err: %w", err)
	}
	log.Info().Msg("Environment variables processed")

	// Log the final configuration
	log.Info().
		Str("database_url", appConfig.DatabaseURL).
		Str("addr", appConfig.Addr).
		Int("max_page_size", appConfig.MaxPageSize).
		Int("db_max_connections", appConfig.DBMaxConnections).
		Int("db_min_connections", appConfig.DBMinConnections).
		Msg("Final configuration loaded")

	return nil
}
