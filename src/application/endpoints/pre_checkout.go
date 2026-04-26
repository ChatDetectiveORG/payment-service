package endpoints

import (
	"time"

	paymentservice "github.com/ChatDetectiveORG/payment-service"
	e "github.com/ChatDetectiveORG/shared/errors"
	h "github.com/ChatDetectiveORG/shared/handlers"
	tele "gopkg.in/telebot.v4"
)

type preCheckoutFilter struct{}

func (f preCheckoutFilter) Filter(update tele.Update) bool {
	return update.PreCheckoutQuery != nil
}

func NewPreCheckoutEndpoint() h.Endpoint {
	ep := h.Endpoint{}
	ep.Init(
		"pre_checkout",
		*h.HandlerChain{}.Init(
			10*time.Second,
			h.InitChainHandler(runPreCheckout, h.EndOnError),
		),
		preCheckoutFilter{},
	)
	return ep
}

func runPreCheckout(update tele.Update, hashe *h.HandlerChainHashe) *e.ErrorInfo {
	return HandlePreCheckout(update)
}

func HandlePreCheckout(update tele.Update) *e.ErrorInfo {
	query := update.PreCheckoutQuery
	if query == nil {
		return e.NewError("precheckout query is nil", "failed to handle precheckout").WithSeverity(e.Notice)
	}
	_, err := paymentservice.MarkPreCheckoutReceived(query.Payload, query.ID)
	return err
}
