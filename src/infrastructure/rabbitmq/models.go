package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Model interface {
	Name() string
	Ensure(ch *amqp.Channel) error
}

func EnsureModels(ch *amqp.Channel, models []Model) error {
	for _, model := range models {
		if model == nil {
			continue
		}
		if err := model.Ensure(ch); err != nil {
			return fmt.Errorf("%s: %w", model.Name(), err)
		}
	}
	return nil
}

type ExchangeModel struct {
	Exchange   string
	Kind       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp.Table
}

func (m ExchangeModel) Name() string { return "exchange:" + m.Exchange }

func (m ExchangeModel) Ensure(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(m.Exchange, m.Kind, m.Durable, m.AutoDelete, m.Internal, m.NoWait, m.Args)
}

type QueueModel struct {
	Queue      string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
}

func (m QueueModel) Name() string { return "queue:" + m.Queue }

func (m QueueModel) Ensure(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(m.Queue, m.Durable, m.AutoDelete, m.Exclusive, m.NoWait, m.Args)
	return err
}

type BindingModel struct {
	Queue      string
	Exchange   string
	RoutingKey string
	NoWait     bool
	Args       amqp.Table
}

func (m BindingModel) Name() string {
	return fmt.Sprintf("binding:%s<=%s[%s]", m.Queue, m.Exchange, m.RoutingKey)
}

func (m BindingModel) Ensure(ch *amqp.Channel) error {
	return ch.QueueBind(m.Queue, m.RoutingKey, m.Exchange, m.NoWait, m.Args)
}
