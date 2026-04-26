package rabbitmq

import (
	"fmt"

	"github.com/ChatDetectiveORG/payment-service/src/infrastructure/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

var RequiredModels = buildRequiredModels()

const shardCount = 64

func buildRequiredModels() []Model {
	models := []Model{
		ExchangeModel{
			Exchange:   "chatdetective.events",
			Kind:       "topic",
			Durable:    true,
			AutoDelete: false,
			Internal:   false,
			NoWait:     false,
			Args:       amqp.Table{},
		},
	}

	for i := 0; i < shardCount; i++ {
		queue := fmt.Sprintf("%s.q%02d", config.PodType, i)
		models = append(models,
			QueueModel{
				Queue:      queue,
				Durable:    true,
				AutoDelete: false,
				Exclusive:  false,
				NoWait:     false,
				Args: amqp.Table{
					"x-single-active-consumer": true,
				},
			},
			BindingModel{
				Queue:      queue,
				Exchange:   "chatdetective.events",
				RoutingKey: queue,
				NoWait:     false,
				Args:       amqp.Table{},
			},
		)
	}

	return models
}
