package config

import (
	"fmt"
	"os"

	env "github.com/pedrobarbosak/go-env-validator"
)

type Config struct {
	LoggerConfig
	HTTPConfig
}

type LoggerConfig struct {
	Level string `env:"LOG_LEVEL,required"`
}

type HTTPConfig struct {
	Host      string `env:"APP_HOST,required"`
	Port      int    `env:"APP_PORT,required"`
	PprofPort int    `env:"PPROF_PORT,required"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := env.UnmarshalFromFile(os.Getenv("CONFIG_PATH"), &cfg); err != nil {
		return nil, fmt.Errorf("load and parse env config: %w", err)
	}

	return &cfg, nil
}
