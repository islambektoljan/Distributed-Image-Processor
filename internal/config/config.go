package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	PostgresURL    string `envconfig:"POSTGRES_URL" default:"postgres://postgres:postgres@127.0.0.1:5433/imagedb?sslmode=disable"`
	RedisURL       string `envconfig:"REDIS_URL" default:"localhost:6379"`
	MinioEndpoint  string `envconfig:"MINIO_ENDPOINT" default:"localhost:9000"`
	MinioAccessKey string `envconfig:"MINIO_ACCESS_KEY" default:"minioadmin"`
	MinioSecretKey string `envconfig:"MINIO_SECRET_KEY" default:"minioadmin"`
	RabbitMQURL    string `envconfig:"RABBITMQ_URL" default:"amqp://guest:guest@localhost:5672/"`
	KeycloakURL    string `envconfig:"KEYCLOAK_URL" default:"http://localhost:8080"`
}

func LoadConfig() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
