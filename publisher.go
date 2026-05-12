package paymentservice

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/ChatDetectiveORG/shared/exports"
	models "github.com/ChatDetectiveORG/shared/postgresModels"
	amqp "github.com/rabbitmq/amqp091-go"
)

// The publisher is purposefully self-contained instead of reusing the consumer's connection holder:
// payment-service is imported by command-handler purely for its EmitPayment helper, and we do not
// want command-handler to spin up a publisher just because it imports the package. Lazy init keeps
// boundaries clean — the connection only opens when ProcessPreCheckout actually publishes.
var (
	publisherMu   sync.Mutex
	publisherConn *amqp.Connection
	publisherCh   *amqp.Channel
)

func getPublisherChannel() (*amqp.Channel, *e.ErrorInfo) {
	publisherMu.Lock()
	defer publisherMu.Unlock()

	if publisherCh != nil && publisherConn != nil && !publisherConn.IsClosed() {
		return publisherCh, e.Nil()
	}

	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		return nil, e.NewError("missing env RABBITMQ_URL", "rabbitmq publisher is not configured").WithSeverity(e.Critical)
	}

	conn, rawErr := amqp.DialConfig(url, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
		Dial:      amqp.DefaultDial(10 * time.Second),
	})
	if rawErr != nil {
		return nil, e.FromError(rawErr, "failed to dial rabbitmq").WithSeverity(e.Critical)
	}

	ch, rawErr := conn.Channel()
	if rawErr != nil {
		_ = conn.Close()
		return nil, e.FromError(rawErr, "failed to open rabbitmq channel").WithSeverity(e.Critical)
	}

	if rawErr := ch.ExchangeDeclare(exports.ExportsExchange, "topic", true, false, false, false, amqp.Table{}); rawErr != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, e.FromError(rawErr, "failed to declare exports exchange").WithSeverity(e.Critical)
	}

	publisherConn = conn
	publisherCh = ch
	return ch, e.Nil()
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

	ch, err := getPublisherChannel()
	if e.IsNonNil(err) {
		return err
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rk := exports.ExportShardRoutingKey(payload.SenderIDHash)
	if rawErr := ch.PublishWithContext(pubCtx, exports.ExportsExchange, rk, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}); rawErr != nil {
		// Drop the cached channel — next call will reopen.
		publisherMu.Lock()
		publisherCh = nil
		publisherMu.Unlock()
		return e.FromError(rawErr, "failed to publish export request").WithSeverity(e.Critical)
	}
	return e.Nil()
}
