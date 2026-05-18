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
	MongoDBConfig
	CassandraConfig
	Neo4jConfig
}

type LoggerConfig struct {
	LogLevel string `env:"LOG_LEVEL,required"`
}

type AppConfig struct {
	AppUserSessionTTLSeconds     int `env:"APP_USER_SESSION_TTL,required"`
	AppLikeTTLSeconds            int `env:"APP_LIKE_TTL,required"`
	AppEventReviewsTTLSeconds    int `env:"APP_EVENT_REVIEWS_TTL,required"`
	AppRecommendationsTTLSeconds int `env:"APP_RECOMMENDATIONS_TTL,required"`
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

type MongoDBConfig struct {
	MongoDBHost     string `env:"MONGODB_HOST,required"`
	MongoDBPort     int    `env:"MONGODB_PORT,required"`
	MongoDBUser     string `env:"MONGODB_USER,required"`
	MongoDBPassword string `env:"MONGODB_PASSWORD,required"`
	MongoDBDatabase string `env:"MONGODB_DATABASE,required"`
}

type CassandraConfig struct {
	CassandraHosts       string `env:"CASSANDRA_HOSTS,required"`
	CassandraPort        int    `env:"CASSANDRA_PORT,required"`
	CassandraUsername    string `env:"CASSANDRA_USERNAME,required"`
	CassandraPassword    string `env:"CASSANDRA_PASSWORD,required"`
	CassandraKeyspace    string `env:"CASSANDRA_KEYSPACE,required"`
	CassandraConsistency string `env:"CASSANDRA_CONSISTENCY,required"`
}

type Neo4jConfig struct {
	Neo4jURL      string `env:"NEO4J_URL,required"`
	Neo4jUser     string `env:"NEO4J_USERNAME,required"`
	Neo4jPassword string `env:"NEO4J_PASSWORD,required"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := env.UnmarshalFromFile(os.Getenv("CONFIG_PATH"), &cfg); err != nil {
		return nil, fmt.Errorf("load and parse env config: %w", err)
	}

	return &cfg, nil
}
