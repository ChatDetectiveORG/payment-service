package rabbitmq

import (
	"sync"
	"time"

	"github.com/ChatDetectiveORG/payment-service/src/infrastructure/config"
	e "github.com/ChatDetectiveORG/shared/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	clientOnce sync.Once
	client     *Client
)

func GetClient(cfg *config.Config) (*Client, *e.ErrorInfo) {
	var initErr *e.ErrorInfo = e.Nil()

	clientOnce.Do(func() {
		if cfg.RabbitMQConfig.URL == "" {
			initErr = e.NewError("missing env RABBITMQ_URL", "rabbitmq url is not configured").WithSeverity(e.Critical)
			return
		}
		c, err := NewClient(cfg.RabbitMQConfig.URL, amqp.Config{
			Heartbeat: 10 * time.Second,
			Locale:    "en_US",
			Dial:      amqp.DefaultDial(10 * time.Second),
		})
		if err != nil {
			initErr = e.FromError(err, "failed to connect to rabbitmq").WithSeverity(e.Critical)
			return
		}
		client = c
	})

	if !initErr.IsNil() {
		return nil, initErr
	}
	return client, e.Nil()
}

func InitRabbitMQ(cfg *config.Config, models []Model) *e.ErrorInfo {
	client, err := GetClient(cfg)
	if !err.IsNil() {
		return err
	}

	ch, rawErr := client.Channel()
	if rawErr != nil {
		return e.FromError(rawErr, "failed to open rabbitmq channel").WithSeverity(e.Critical)
	}
	defer func() { _ = ch.Close() }()

	if rawErr := EnsureModels(ch, models); rawErr != nil {
		return e.FromError(rawErr, "failed to ensure rabbitmq models").WithSeverity(e.Critical)
	}
	return e.Nil()
}

func NewRabbitmqChannel(cfg *config.Config) (*amqp.Channel, *e.ErrorInfo) {
	client, err := GetClient(cfg)
	if !err.IsNil() {
		return nil, err
	}
	ch, rawErr := client.Channel()
	if rawErr != nil {
		return nil, e.FromError(rawErr, "failed to get rabbitmq channel")
	}
	return ch, e.Nil()
}
