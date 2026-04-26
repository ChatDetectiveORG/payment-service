package paymentservice

import "time"

type PaymentType string

const (
	PaymentTypeLevelUp    PaymentType = "levelUp"
	PaymentTypeExportChat PaymentType = "ExportChat"
)

type PaymentOpts struct {
	Recipient  *PaymentRecipientOpts
	Invoice    *PaymentInvoiceOpts
	LevelUp    *LevelUpOpts
	ExportChat *ExportChatOpts

	// Timeout is treated as a deadline by EmitPayment.
	Timeout *time.Time
}

type PaymentRecipientOpts struct {
	TelegramUserID int64
	ChatID         int64
}

type PaymentInvoiceOpts struct {
	Title       string
	Description string
	PriceLabel  string
}

type LevelUpOpts struct {
	Levels int `json:"levels"`
}

type ExportChatOpts struct {
	Messages int `json:"messages"`
}

type PaymentMethod struct {
	Code          string
	ButtonText    string
	Currency      string
	ProviderToken string
	StarsRate     float64
}
