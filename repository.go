package paymentservice

import (
	"encoding/json"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	models "github.com/ChatDetectiveORG/shared/postgresModels"
	"github.com/go-pg/pg/v10"
)

type paymentServiceMetadata struct {
	ChatID int64 `json:"chat_id,omitempty"`

	LevelUp *LevelUpOpts `json:"level_up,omitempty"`
	Mirror  *MirrorOpts  `json:"mirror,omitempty"`
}

func getClientByTelegramID(tgUserID int64) (*models.Telegramuser, *e.ErrorInfo) {
	user := &models.Telegramuser{}
	if err := user.GetByTelegramID(GetDB(), tgUserID); e.IsNonNil(err) {
		return nil, err
	}
	return user, e.Nil()
}

func createInvoicePayment(user *models.Telegramuser, paymentType PaymentType, payload string, amount int, method PaymentMethod, opts *PaymentOpts) (*models.Payment, *e.ErrorInfo) {
	metadata, err := marshalPaymentMetadata(paymentType, opts)
	if e.IsNonNil(err) {
		return nil, err
	}

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
		InvoiceTitle:       opts.Invoice.Title,
		InvoiceDescription: opts.Invoice.Description,
		PriceLabel:         opts.Invoice.PriceLabel,
		ServiceMetadata:    metadata,
		Status:             models.PaymentStatusInvoiceSent,
	}
	if _, err := GetDB().Model(payment).Insert(); err != nil {
		return nil, e.FromError(err, "failed to create payment").WithSeverity(e.Notice)
	}
	return payment, e.Nil()
}

func marshalPaymentMetadata(paymentType PaymentType, opts *PaymentOpts) (string, *e.ErrorInfo) {
	metadata := paymentServiceMetadata{}
	if opts != nil && opts.Recipient != nil {
		metadata.ChatID = opts.Recipient.ChatID
	}
	switch paymentType {
	case PaymentTypeLevelUp:
		metadata.LevelUp = opts.LevelUp
	case PaymentTypeMirror:
		metadata.Mirror = opts.Mirror
	default:
		return "", e.Nil()
	}

	raw, err := json.Marshal(metadata)
	if err != nil {
		return "", e.FromError(err, "failed to marshal payment metadata").WithSeverity(e.Notice)
	}
	return string(raw), e.Nil()
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

func parsePaymentMetadata(payment *models.Payment) (*paymentServiceMetadata, *e.ErrorInfo) {
	metadata := &paymentServiceMetadata{}
	if payment.ServiceMetadata == "" {
		return metadata, e.Nil()
	}
	if err := json.Unmarshal([]byte(payment.ServiceMetadata), metadata); err != nil {
		return nil, e.FromError(err, "failed to unmarshal payment metadata").WithSeverity(e.Notice)
	}
	return metadata, e.Nil()
}

func grantMirrorPurchase(payment *models.Payment, now time.Time) (*paymentServiceMetadata, *e.ErrorInfo) {
	metadata, err := parsePaymentMetadata(payment)
	if e.IsNonNil(err) {
		return nil, err
	}
	if metadata.Mirror == nil || metadata.Mirror.PendingMirrorID <= 0 {
		return nil, e.NewError("mirror metadata is invalid", "failed to grant mirror purchase").WithSeverity(e.Notice)
	}

	if _, err := models.ActivateMirror(GetDB(), metadata.Mirror.PendingMirrorID, &payment.ID, now); e.IsNonNil(err) {
		return nil, err
	}
	return metadata, e.Nil()
}

func grantLevelPurchase(payment *models.Payment, now time.Time) (*paymentServiceMetadata, *e.ErrorInfo) {
	metadata, err := parsePaymentMetadata(payment)
	if e.IsNonNil(err) {
		return nil, err
	}
	if metadata.LevelUp == nil || metadata.LevelUp.Levels <= 0 {
		return nil, e.NewError("levelUp metadata is invalid", "failed to grant level purchase").WithSeverity(e.Notice)
	}

	level := &models.UserLevels{
		LinkedUserID:    payment.ClientID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Level:           metadata.LevelUp.Levels,
		UntilTimestamp:  now.AddDate(0, 1, 0).Unix(),
		SourcePaymentID: &payment.ID,
	}
	if _, rawErr := GetDB().Model(level).OnConflict("(source_payment_id) DO NOTHING").Insert(); rawErr != nil {
		return nil, e.FromError(rawErr, "failed to grant user levels").WithSeverity(e.Notice)
	}
	return metadata, e.Nil()
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
