package paymentservice

import (
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	models "github.com/ChatDetectiveORG/shared/postgresModels"
	"github.com/google/uuid"
	tele "gopkg.in/telebot.v4"
)

const defaultPaymentWait = 2 * time.Minute

func EmitPayment(paymentType *PaymentType, opts *PaymentOpts) (*e.ErrorInfo, int) {
	if paymentType == nil {
		return e.NewError("payment type is required", "failed to emit payment").WithSeverity(e.Notice), 0
	}
	if opts == nil {
		return e.NewError("payment opts are required", "failed to emit payment").WithSeverity(e.Notice), 0
	}
	if err := validateRecipient(opts.Recipient); e.IsNonNil(err) {
		return err, 0
	}
	if err := validateInvoice(opts.Invoice); e.IsNonNil(err) {
		return err, 0
	}

	client, err := getClientByTelegramID(opts.Recipient.TelegramUserID)
	if e.IsNonNil(err) {
		return err, 0
	}

	baseStars, err := calculatePrice(*paymentType, opts)
	if e.IsNonNil(err) {
		return err, 0
	}

	if len(availablePaymentMethods) == 0 {
		sendInformationalMessage(opts.Recipient.ChatID, defaultNoPaymentMethodsText)
		return e.NewError("payment methods are not configured", "failed to emit payment").WithSeverity(e.Notice), 0
	}

	method := availablePaymentMethods[0]
	amount := convertStarsAmount(baseStars, method)
	payload := uuid.New().String()

	payment, err := createInvoicePayment(client, *paymentType, payload, amount, method, opts.Invoice)
	if e.IsNonNil(err) {
		return err, 0
	}

	if err := sendInvoice(opts.Recipient.ChatID, payload, amount, method, opts.Invoice); e.IsNonNil(err) {
		return err, payment.ID
	}

	if err := waitForPreCheckout(payment.ID, paymentDeadline(opts.Timeout)); e.IsNonNil(err) {
		_ = markPaymentTimedOut(payment.ID)
		sendInformationalMessage(opts.Recipient.ChatID, defaultPaymentTimeoutText)
		return err, payment.ID
	}

	return e.Nil(), payment.ID
}

func ApprovePreCheckout(paymentID int) *e.ErrorInfo {
	payment, err := getPaymentByID(paymentID)
	if e.IsNonNil(err) {
		return err
	}
	if payment.PreCheckoutID == "" {
		return e.NewError("precheckout id is empty", "failed to approve payment").WithSeverity(e.Notice)
	}

	bot, err := GetBot()
	if e.IsNonNil(err) {
		return err
	}
	if rawErr := bot.Accept(&tele.PreCheckoutQuery{ID: payment.PreCheckoutID}); rawErr != nil {
		return e.FromError(rawErr, "failed to approve precheckout").WithSeverity(e.Notice)
	}
	return markPaymentApproved(payment)
}

func CancelPreCheckout(paymentID int, userSideError string) *e.ErrorInfo {
	payment, err := getPaymentByID(paymentID)
	if e.IsNonNil(err) {
		return err
	}
	if payment.PreCheckoutID == "" {
		return e.NewError("precheckout id is empty", "failed to cancel payment").WithSeverity(e.Notice)
	}
	if userSideError == "" {
		userSideError = defaultPreCheckoutCancelText
	}

	bot, err := GetBot()
	if e.IsNonNil(err) {
		return err
	}
	if rawErr := bot.Accept(&tele.PreCheckoutQuery{ID: payment.PreCheckoutID}, userSideError); rawErr != nil {
		return e.FromError(rawErr, "failed to cancel precheckout").WithSeverity(e.Notice)
	}
	return markPaymentCancelled(payment, userSideError)
}

func validateRecipient(opts *PaymentRecipientOpts) *e.ErrorInfo {
	if opts == nil {
		return e.NewError("recipient opts are required", "failed to emit payment").WithSeverity(e.Notice)
	}
	if opts.TelegramUserID == 0 {
		return e.NewError("telegram user id is required", "failed to emit payment").WithSeverity(e.Notice)
	}
	if opts.ChatID == 0 {
		return e.NewError("chat id is required", "failed to emit payment").WithSeverity(e.Notice)
	}
	return e.Nil()
}

func validateInvoice(opts *PaymentInvoiceOpts) *e.ErrorInfo {
	if opts == nil {
		return e.NewError("invoice opts are required", "failed to emit payment").WithSeverity(e.Notice)
	}
	if opts.Title == "" {
		return e.NewError("invoice title is required", "failed to emit payment").WithSeverity(e.Notice)
	}
	if opts.Description == "" {
		return e.NewError("invoice description is required", "failed to emit payment").WithSeverity(e.Notice)
	}
	if opts.PriceLabel == "" {
		return e.NewError("invoice price label is required", "failed to emit payment").WithSeverity(e.Notice)
	}
	return e.Nil()
}

func paymentDeadline(timeout *time.Time) time.Time {
	if timeout == nil || timeout.IsZero() {
		return time.Now().Add(defaultPaymentWait)
	}
	return *timeout
}

func waitForPreCheckout(paymentID int, deadline time.Time) *e.ErrorInfo {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		if time.Now().After(deadline) {
			return e.NewError("payment precheckout timeout", "failed to emit payment").WithSeverity(e.Notice)
		}

		payment, err := getPaymentByID(paymentID)
		if e.IsNonNil(err) {
			return err
		}
		if payment.Status == models.PaymentStatusPreCheckoutReceived || payment.Status == models.PaymentStatusApproved {
			return e.Nil()
		}

		<-ticker.C
	}
}

func sendInvoice(chatID int64, payload string, amount int, method PaymentMethod, invoice *PaymentInvoiceOpts) *e.ErrorInfo {
	bot, err := GetBot()
	if e.IsNonNil(err) {
		return err
	}
	_, rawErr := bot.Send(&tele.Chat{ID: chatID}, &tele.Invoice{
		Title:       invoice.Title,
		Description: invoice.Description,
		Payload:     payload,
		Currency:    method.Currency,
		Token:       method.ProviderToken,
		Prices: []tele.Price{
			{
				Label:  invoice.PriceLabel,
				Amount: amount,
			},
		},
	})
	if rawErr != nil {
		return e.FromError(rawErr, "failed to send payment invoice").WithSeverity(e.Notice)
	}
	return e.Nil()
}

func sendInformationalMessage(chatID int64, text string) {
	bot, err := GetBot()
	if e.IsNonNil(err) {
		return
	}
	_, _ = bot.Send(&tele.Chat{ID: chatID}, text)
}
