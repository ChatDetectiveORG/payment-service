package config

import (
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/spf13/viper"
)

const PodType = "shipping"

type Config struct {
	RuntimeConfig  *RuntimeConfig
	RabbitMQConfig *RabbitMQConfig
	PostgresConfig *PostgresConfig
}

type RuntimeConfig struct {
	NumRoutingGorutines int
	PodID               string
	PodType             string
	HandlerLiveDuration time.Duration
}

type RabbitMQConfig struct {
	URL string
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func FetchConfig() (*Config, *e.ErrorInfo) {
	viper.AutomaticEnv()

	cfg := &Config{
		PostgresConfig: &PostgresConfig{
			Host:     viper.GetString("POSTGRES_HOST"),
			Port:     viper.GetString("POSTGRES_PORT"),
			User:     viper.GetString("POSTGRES_USER"),
			Password: viper.GetString("POSTGRES_PASSWORD"),
			Database: viper.GetString("POSTGRES_DB"),
		},
		RabbitMQConfig: &RabbitMQConfig{
			URL: viper.GetString("RABBITMQ_URL"),
		},
		RuntimeConfig: &RuntimeConfig{
			NumRoutingGorutines: viper.GetInt("NUM_ROUTING_GOROUTINES"),
			PodID:               viper.GetString("POD_ID"),
			PodType:             PodType,
			HandlerLiveDuration: time.Minute * 10,
		},
	}

	return cfg, e.Nil()
}
