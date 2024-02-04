package config

import (
	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v9"
	"os"
)

const (
	// AppName app name
	AppName = "url-shortener"
	// configPathEnvName is a name of env variable which contains the path to config file
	configPathEnvName = "CONFIG_FILE_PATH"

	// defaultConfigFileAddress default configuration file address
	defaultConfigFileAddress = "./config.toml"
)

type (
	// Config is app config
	Config struct {
		// commitHash is a git commit hash of this app build
		commitHash string
		// tag is a git tag of this app build
		tag string

		// ServiceName logging service name
		ServiceName string `toml:"service_name" validate:"required"`
		// Environment development, staging, production
		Environment string `toml:"environment" validate:"required"`
		// HTTPAddr internal server http address
		HTTPAddr string `toml:"http_addr" validate:"required"`
		// LogLevel ...
		LogLevel string `toml:"log_level" validate:"required"`
		// EnableApplicationProfiler ...
		EnableApplicationProfiler bool `toml:"enable_application_profiler"`
		// Coordination ...
		Coordination Coordination `toml:"coordination" validate:"required,dive"`
		// MySqlDB application DB config
		AppDB MySqlDB `toml:"app_db" validate:"required,dive"`
	}

	Coordination struct {
		// NodeID represents the unique identifier for the application, which must maintain its uniqueness within the cluster.
		NodeID string `toml:"node_id" validation:"required"`
		// RangeSize range size to be reserved in each request
		RangeSize int `toml:"range_size" validation:"required"`
		// DataStore MySql datastore that is being used for coordination purposes
		DataStore MySqlDB `toml:"datastore" validate:"required,dive"`
	}

	// MySqlDB contains mysql db configs
	MySqlDB struct {
		// Credentials ...
		Credential DBCredential `toml:"credential" validate:"required"`

		// Toml config constraint https://github.com/toml-lang/toml/issues/514
		// ReadTimeoutSec ...
		ReadTimeoutSec int `toml:"read_timeout_sec" validate:"required"`
		// WriteTimeoutSec ...
		WriteTimeoutSec int `toml:"write_timeout_sec" validate:"required"`
		// MaxOpenConn ...
		MaxOpenConn int `toml:"max_open_conn" validate:"required"`
		// ConnLifetimeSec ...
		ConnLifetimeSec int `toml:"connection_lifetime_sec" validate:"required"`
	}

	// DBCredential database credential
	DBCredential struct {
		// Host ...
		Host string `toml:"host" validate:"required"`
		// DBName ...
		DBName string `toml:"db_name" validate:"required"`
		// Username ...
		Username string `toml:"username" validate:"required"`
		// Password ...
		Password string `toml:"password" validate:"required"`
	}
)

// CommitHash returns git commit hash of this app build
func (c *Config) CommitHash() string {
	return c.commitHash
}

// Tag returns git tag of this app build
func (c *Config) Tag() string {
	return c.tag
}

// GetUserAgent creates user-agent string for remote calls
func (c *Config) GetUserAgent() string {
	return AppName + "/" + c.Tag()
}

// Load reads the config file path from environment variable and calls LoadFromPath
func Load(commitHash string, tag string) (*Config, error) {
	configFilePath := os.Getenv(configPathEnvName)
	if configFilePath == "" {
		// TODO: Warning log "env variable is not defined"
		configFilePath = defaultConfigFileAddress
	}

	cfg, err := LoadFromPath(configFilePath, commitHash, tag)
	if err != nil {
		return nil, errors.Wrap(err, "can not load config from file")
	}

	return cfg, nil
}

// LoadFromPath loads configurations
// commitHash is a git commit hash of this app build
// tag is a git tag of this app build
func LoadFromPath(configFilePath, commitHash, tag string) (*Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(configFilePath, &cfg)
	if err != nil {
		return nil, errors.Wrap(err, "can not unmarshal config data")
	}

	cfg.commitHash = commitHash
	cfg.tag = tag

	// validating app file configs
	v := validator.New()
	err = v.Struct(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "config file is not valid")
	}
	return &cfg, nil
}
