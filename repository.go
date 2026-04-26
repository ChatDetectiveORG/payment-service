package paymentservice

import (
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	models "github.com/ChatDetectiveORG/shared/postgresModels"
	"github.com/go-pg/pg/v10"
)

func getClientByTelegramID(tgUserID int64) (*models.Telegramuser, *e.ErrorInfo) {
	user := &models.Telegramuser{}
	if err := user.GetByTelegramID(GetDB(), tgUserID); e.IsNonNil(err) {
		return nil, err
	}
	return user, e.Nil()
}

func createInvoicePayment(user *models.Telegramuser, paymentType PaymentType, payload string, amount int, method PaymentMethod, invoice *PaymentInvoiceOpts) (*models.Payment, *e.ErrorInfo) {
	now := time.Now()
	payment := &models.Payment{
		ClientID:           user.ID,
		Client:             user,
		CreatedAt:          now,
		UpdatedAt:          now,
		IsSuccess:          false,
		ServiceType:        string(paymentType),
		Payload:            payload,
		Currency:           method.Currency,
		Amount:             amount,
		PaymentMethod:      method.Code,
		InvoiceTitle:       invoice.Title,
		InvoiceDescription: invoice.Description,
		PriceLabel:         invoice.PriceLabel,
		Status:             models.PaymentStatusInvoiceSent,
	}
	if _, err := GetDB().Model(payment).Insert(); err != nil {
		return nil, e.FromError(err, "failed to create payment").WithSeverity(e.Notice)
	}
	return payment, e.Nil()
}

func getPaymentByID(paymentID int) (*models.Payment, *e.ErrorInfo) {
	payment := &models.Payment{ID: paymentID}
	if err := GetDB().Model(payment).WherePK().Select(); err != nil {
		return nil, e.FromError(err, "failed to get payment").WithSeverity(e.Notice)
	}
	return payment, e.Nil()
}

func getPaymentByPayload(payload string) (*models.Payment, *e.ErrorInfo) {
	payment := &models.Payment{}
	if err := GetDB().Model(payment).Where("payload = ?", payload).Select(); err != nil {
		return nil, e.FromError(err, "failed to get payment by payload").WithSeverity(e.Notice)
	}
	return payment, e.Nil()
}

func markPreCheckoutReceived(payload string, preCheckoutID string) (*models.Payment, *e.ErrorInfo) {
	payment, err := getPaymentByPayload(payload)
	if e.IsNonNil(err) {
		return nil, err
	}
	payment.PreCheckoutID = preCheckoutID
	payment.Status = models.PaymentStatusPreCheckoutReceived
	payment.UpdatedAt = time.Now()
	if _, rawErr := GetDB().Model(payment).WherePK().Column("pre_checkout_id", "status", "updated_at").Update(); rawErr != nil {
		return nil, e.FromError(rawErr, "failed to update payment precheckout").WithSeverity(e.Notice)
	}
	return payment, e.Nil()
}

func MarkPreCheckoutReceived(payload string, preCheckoutID string) (*models.Payment, *e.ErrorInfo) {
	return markPreCheckoutReceived(payload, preCheckoutID)
}

func markPaymentTimedOut(paymentID int) *e.ErrorInfo {
	payment := &models.Payment{ID: paymentID, Status: models.PaymentStatusTimedOut, UpdatedAt: time.Now()}
	_, err := GetDB().Model(payment).WherePK().Column("status", "updated_at").Update()
	if err != nil && err != pg.ErrNoRows {
		return e.FromError(err, "failed to mark payment timed out").WithSeverity(e.Notice)
	}
	return e.Nil()
}

func markPaymentApproved(payment *models.Payment) *e.ErrorInfo {
	payment.IsSuccess = true
	payment.Status = models.PaymentStatusApproved
	payment.UpdatedAt = time.Now()
	_, err := GetDB().Model(payment).WherePK().Column("is_success", "status", "updated_at").Update()
	if err != nil {
		return e.FromError(err, "failed to approve payment").WithSeverity(e.Notice)
	}
	return e.Nil()
}

func markPaymentCancelled(payment *models.Payment, userSideError string) *e.ErrorInfo {
	payment.IsSuccess = false
	payment.Status = models.PaymentStatusCancelled
	payment.UserSideError = userSideError
	payment.UpdatedAt = time.Now()
	_, err := GetDB().Model(payment).WherePK().Column("is_success", "status", "user_side_error", "updated_at").Update()
	if err != nil {
		return e.FromError(err, "failed to cancel payment").WithSeverity(e.Notice)
	}
	return e.Nil()
}
