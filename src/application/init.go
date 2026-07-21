package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/ChatDetectiveORG/payment-service/src/application/endpoints"
	"github.com/ChatDetectiveORG/payment-service/src/infrastructure/config"
	"github.com/ChatDetectiveORG/payment-service/src/infrastructure/rabbitmq"
	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/ChatDetectiveORG/shared/amqputil"
	amqp "github.com/rabbitmq/amqp091-go"
	tele "gopkg.in/telebot.v4"
)

var errors = make(chan *e.ErrorInfo, 1000)

const shardCount = 64

func ListenToRabbitmq(cfg *config.Config, ctx context.Context, wg *sync.WaitGroup) *e.ErrorInfo {
	go handleErrors(errors, ctx, wg)

	wg.Add(1)
	go func() {
		defer wg.Done()
		amqputil.RunConsumerLoop(ctx, amqputil.ConsumerConfig{
			Dial: func() (*amqputil.ConsumerSession, error) {
				deliveries, tags, ch, dialErr := initRabbitmqQueue(cfg)
				if !dialErr.IsNil() {
					return nil, dialErr
				}
				return &amqputil.ConsumerSession{
					Deliveries: deliveries,
					Channel:    ch,
					Cleanup: func() {
						for _, tag := range tags {
							_ = ch.Cancel(tag, false)
						}
						_ = ch.Close()
					},
				}, nil
			},
			OnDelivery: func(delivery amqp.Delivery) {
				if err := handleDelivery(delivery); !err.IsNil() {
					errors <- err.WithData(map[string]any{"rk": delivery.RoutingKey}).WithSeverity(e.Critical)
					if nackErr := delivery.Nack(false, false); nackErr != nil {
						errors <- e.FromError(nackErr, "failed to nack delivery").WithSeverity(e.Critical)
					}
					return
				}
				if ackErr := delivery.Ack(false); ackErr != nil {
					errors <- e.FromError(ackErr, "failed to ack delivery")
				}
			},
		})
	}()

	return e.Nil()
}

func initRabbitmqQueue(cfg *config.Config) (<-chan amqp.Delivery, []string, *amqp.Channel, *e.ErrorInfo) {
	rabbitmqChannel, err := rabbitmq.NewRabbitmqChannel(cfg)
	if !err.IsNil() {
		return nil, nil, nil, err
	}
	_ = rabbitmqChannel.Qos(50, 0, false)

	merged := make(chan amqp.Delivery, 1000)
	consumerTags := make([]string, 0, shardCount)
	var forwardWg sync.WaitGroup
	forwardWg.Add(shardCount)

	podID := cfg.RuntimeConfig.PodID
	if podID == "" {
		podID = os.Getenv("POD_ID")
	}
	if podID == "" {
		podID = "unknown"
	}

	for i := 0; i < shardCount; i++ {
		queue := fmtShardQueue(i)
		tag := fmt.Sprintf("payments-%s-%s", podID, queue)
		consumerTags = append(consumerTags, tag)

		consumer, rawErr := rabbitmqChannel.Consume(queue, tag, false, false, false, false, amqp.Table{})
		if rawErr != nil {
			_ = rabbitmqChannel.Close()
			return nil, nil, nil, e.FromError(rawErr, "failed to init consumer").WithSeverity(e.Critical).WithData(map[string]any{"queue": queue})
		}

		go func(c <-chan amqp.Delivery) {
			defer forwardWg.Done()
			for delivery := range c {
				merged <- delivery
			}
		}(consumer)
	}

	go func() {
		forwardWg.Wait()
		close(merged)
	}()

	return merged, consumerTags, rabbitmqChannel, e.Nil()
}

func handleDelivery(delivery amqp.Delivery) *e.ErrorInfo {
	var update tele.Update
	if err := json.Unmarshal(delivery.Body, &update); err != nil {
		return e.FromError(err, "unmarshal update").WithSeverity(e.Critical)
	}
	if update.PreCheckoutQuery == nil {
		log.Printf("trace=%s ignored non-precheckout shipping update", delivery.CorrelationId)
		return e.Nil()
	}
	log.Printf(
		"trace=%s received precheckout id=%s payload=%s",
		delivery.CorrelationId,
		update.PreCheckoutQuery.ID,
		update.PreCheckoutQuery.Payload,
	)
	mirrorID, _ := delivery.Headers["mirror_id"].(string)
	return endpoints.HandlePreCheckout(update, mirrorID)
}

func handleErrors(src chan *e.ErrorInfo, ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-src:
			log.Println(err.JSON())
		}
	}
}

func fmtShardQueue(i int) string {
	return fmt.Sprintf("%s.q%02d", config.PodType, i)
}
