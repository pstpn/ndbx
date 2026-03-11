package config

import (
	"fmt"
	"os"

	env "github.com/pedrobarbosak/go-env-validator"
)

type Config struct {
	LoggerConfig
	AppConfig
	HTTPConfig
	RedisConfig
}

type LoggerConfig struct {
	LogLevel string `env:"LOG_LEVEL,required"`
}

type AppConfig struct {
	AppUserSessionTTLSeconds int `env:"APP_USER_SESSION_TTL,required"`
}

type HTTPConfig struct {
	HTTPHost  string `env:"APP_HOST,required"`
	HTTPPort  int    `env:"APP_PORT,required"`
	PprofPort int    `env:"PPROF_PORT,required"`
}

type RedisConfig struct {
	RedisHost     string `env:"REDIS_HOST,required"`
	RedisPort     int    `env:"REDIS_PORT,required"`
	RedisPassword string `env:"REDIS_PASSWORD,required"`
	RedisDB       int    `env:"REDIS_DB,required"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := env.UnmarshalFromFile(os.Getenv("CONFIG_PATH"), &cfg); err != nil {
		return nil, fmt.Errorf("load and parse env config: %w", err)
	}

	return &cfg, nil
}
