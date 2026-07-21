package paymentservice

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/ChatDetectiveORG/shared/amqputil"
	"github.com/ChatDetectiveORG/shared/exports"
	models "github.com/ChatDetectiveORG/shared/postgresModels"
	amqp "github.com/rabbitmq/amqp091-go"
)

// The publisher is purposefully self-contained instead of reusing the consumer's connection holder:
// payment-service is imported by command-handler purely for its EmitPayment helper, and we do not
// want command-handler to spin up a publisher just because it imports the package. Lazy init keeps
// boundaries clean — the connection only opens when ProcessPreCheckout actually publishes.
var (
	exportPublisherMu sync.Mutex
	exportPublisher   *amqputil.PublishChannel
)

func getExportPublisher() (*amqputil.PublishChannel, *e.ErrorInfo) {
	exportPublisherMu.Lock()
	defer exportPublisherMu.Unlock()

	if exportPublisher != nil {
		return exportPublisher, e.Nil()
	}

	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		return nil, e.NewError("missing env RABBITMQ_URL", "rabbitmq publisher is not configured").WithSeverity(e.Critical)
	}

	open := func() (*amqp.Channel, error) {
		conn, rawErr := amqp.DialConfig(url, amqp.Config{
			Heartbeat: 10 * time.Second,
			Locale:    "en_US",
			Dial:      amqp.DefaultDial(10 * time.Second),
		})
		if rawErr != nil {
			return nil, rawErr
		}

		ch, rawErr := conn.Channel()
		if rawErr != nil {
			_ = conn.Close()
			return nil, rawErr
		}

		if rawErr := ch.ExchangeDeclare(exports.ExportsExchange, "topic", true, false, false, false, amqp.Table{}); rawErr != nil {
			_ = ch.Close()
			_ = conn.Close()
			return nil, rawErr
		}
		return ch, nil
	}

	ch, rawErr := open()
	if rawErr != nil {
		return nil, e.FromError(rawErr, "failed to open rabbitmq export publisher").WithSeverity(e.Critical)
	}

	exportPublisher = amqputil.NewPublishChannel(ch, open)
	return exportPublisher, e.Nil()
}

func publishExportRequest(payment *models.Payment, metadata *paymentServiceMetadata) *e.ErrorInfo {
	if metadata == nil || metadata.ExportChat == nil {
		return e.NewError("exportChat metadata is missing", "failed to publish export request").WithSeverity(e.Notice)
	}

	payload := exports.ExportRequest{
		PaymentID:        payment.ID,
		SenderIDHash:     metadata.ExportChat.SenderIDHash,
		InterlocutorCode: metadata.ExportChat.InterlocutorCode,
		StatusChatID:     metadata.ExportChat.StatusChatID,
		MessagesCount:    metadata.ExportChat.Messages,
	}

	body, rawErr := json.Marshal(payload)
	if rawErr != nil {
		return e.FromError(rawErr, "failed to marshal export request").WithSeverity(e.Notice)
	}

	pub, err := getExportPublisher()
	if e.IsNonNil(err) {
		return err
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rk := exports.ExportShardRoutingKey(payload.SenderIDHash)
	if rawErr := pub.Publish(pubCtx, exports.ExportsExchange, rk, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); rawErr != nil {
		exportPublisherMu.Lock()
		exportPublisher = nil
		exportPublisherMu.Unlock()
		return e.FromError(rawErr, "failed to publish export request").WithSeverity(e.Critical)
	}
	return e.Nil()
}
