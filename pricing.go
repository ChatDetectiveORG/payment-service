package paymentservice

import (
	"math"

	e "github.com/ChatDetectiveORG/shared/errors"
	tele "gopkg.in/telebot.v4"
)

const (
	levelUpPriceStars            = 1
	defaultPaymentMethodStars    = "telegram_stars"
	defaultNoPaymentMethodsText  = "Сейчас оплату провести не получится. Попробуйте позже."
	defaultPaymentTimeoutText    = "Не дождались подтверждения оплаты. Попробуйте ещё раз."
	defaultPreCheckoutCancelText = "Не удалось подтвердить оплату. Попробуйте позже."
)

var availablePaymentMethods = []PaymentMethod{
	{
		Code:          defaultPaymentMethodStars,
		ButtonText:    "Звёздами",
		Currency:      tele.Stars,
		ProviderToken: "",
		StarsRate:     1,
	},
}

func calculatePrice(paymentType PaymentType, opts *PaymentOpts) (int, *e.ErrorInfo) {
	switch paymentType {
	case PaymentTypeLevelUp:
		if opts == nil || opts.LevelUp == nil {
			return 0, e.NewError("levelUp opts are required", "failed to calculate payment price").WithSeverity(e.Notice)
		}
		if opts.LevelUp.Levels <= 0 {
			return 0, e.NewError("levels must be positive", "failed to calculate payment price").WithSeverity(e.Notice)
		}
		return opts.LevelUp.Levels * levelUpPriceStars, e.Nil()
	default:
		return 0, e.NewError("unsupported payment type", "failed to calculate payment price").WithSeverity(e.Notice)
	}
}

func convertStarsAmount(stars int, method PaymentMethod) int {
	if method.StarsRate <= 0 {
		return stars
	}
	return int(math.Ceil(float64(stars) * method.StarsRate))
}
