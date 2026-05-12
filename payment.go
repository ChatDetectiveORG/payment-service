package paymentservice

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	models "github.com/ChatDetectiveORG/shared/postgresModels"
	utils "github.com/ChatDetectiveORG/shared/utils"
	"github.com/google/uuid"
	tele "gopkg.in/telebot.v4"
)

const levelPurchaseBotMention = "@MajorFanOfInnokentii_bot"

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
	if opts.MirrorID != "" {
		sendInformationalMessage(opts.Recipient.ChatID, mirrorPaymentOnlyMainBotText, opts.MirrorID)
		return e.NewError("payments are disabled for mirrors", "failed to emit payment").WithSeverity(e.Notice), 0
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
		sendInformationalMessage(opts.Recipient.ChatID, defaultNoPaymentMethodsText, "")
		return e.NewError("payment methods are not configured", "failed to emit payment").WithSeverity(e.Notice), 0
	}

	method := availablePaymentMethods[0]
	amount := convertStarsAmount(baseStars, method)
	payload := uuid.New().String()

	payment, err := createInvoicePayment(client, *paymentType, payload, amount, method, opts)
	if e.IsNonNil(err) {
		return err, 0
	}

	if err := sendInvoice(opts.Recipient.ChatID, payload, amount, method, opts.Invoice, ""); e.IsNonNil(err) {
		return err, payment.ID
	}

	return e.Nil(), payment.ID
}

func ApprovePreCheckout(paymentID int) *e.ErrorInfo {
	return approvePreCheckout(paymentID, "")
}

func approvePreCheckout(paymentID int, mirrorID string) *e.ErrorInfo {
	payment, err := getPaymentByID(paymentID)
	if e.IsNonNil(err) {
		return err
	}
	if payment.PreCheckoutID == "" {
		return e.NewError("precheckout id is empty", "failed to approve payment").WithSeverity(e.Notice)
	}

	bot, err := GetBotByMirrorID(mirrorID)
	if e.IsNonNil(err) {
		return err
	}
	if rawErr := bot.Accept(&tele.PreCheckoutQuery{ID: payment.PreCheckoutID}); rawErr != nil {
		return e.FromError(rawErr, "failed to approve precheckout").WithSeverity(e.Notice)
	}
	return markPaymentApproved(payment)
}

func ProcessPreCheckout(payload string, preCheckoutID string, mirrorID string) *e.ErrorInfo {
	log.Printf("processing precheckout payload=%s id=%s", payload, preCheckoutID)
	payment, err := markPreCheckoutReceived(payload, preCheckoutID)
	if e.IsNonNil(err) {
		return err
	}
	if mirrorID != "" {
		_ = cancelPreCheckout(payment.ID, mirrorPaymentOnlyMainBotText, mirrorID)
		return e.Nil()
	}

	var metadata *paymentServiceMetadata
	switch PaymentType(payment.ServiceType) {
	case PaymentTypeLevelUp:
		metadata, err = grantLevelPurchase(payment, time.Now())
		if e.IsNonNil(err) {
			_ = CancelPreCheckout(payment.ID, defaultPreCheckoutCancelText)
			return err
		}
	case PaymentTypeMirror:
		metadata, err = grantMirrorPurchase(payment, time.Now())
		if e.IsNonNil(err) {
			_ = CancelPreCheckout(payment.ID, defaultPreCheckoutCancelText)
			return err
		}
	case PaymentTypeExportChat:
		metadata, err = grantExportChatPurchase(payment)
		if e.IsNonNil(err) {
			_ = CancelPreCheckout(payment.ID, defaultPreCheckoutCancelText)
			return err
		}
	default:
		_ = CancelPreCheckout(payment.ID, defaultPreCheckoutCancelText)
		return e.NewError("unsupported payment type", "failed to process precheckout").WithSeverity(e.Notice)
	}

	if err := ApprovePreCheckout(payment.ID); e.IsNonNil(err) {
		return err
	}
	if metadata != nil && metadata.LevelUp != nil {
		if err := sendLevelPurchaseSuccessMessage(payment, metadata); e.IsNonNil(err) {
			log.Printf("failed to send level purchase success message payment_id=%d: %s", payment.ID, err.JSON())
		}
	}
	if metadata != nil && metadata.Mirror != nil {
		if err := sendMirrorPurchaseSuccessMessage(payment, metadata); e.IsNonNil(err) {
			log.Printf("failed to send mirror purchase success message payment_id=%d: %s", payment.ID, err.JSON())
		}
	}
	log.Printf("approved precheckout payment_id=%d payload=%s", payment.ID, payload)
	return e.Nil()
}

func CancelPreCheckout(paymentID int, userSideError string) *e.ErrorInfo {
	return cancelPreCheckout(paymentID, userSideError, "")
}

func cancelPreCheckout(paymentID int, userSideError string, mirrorID string) *e.ErrorInfo {
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

	bot, err := GetBotByMirrorID(mirrorID)
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

func sendInvoice(chatID int64, payload string, amount int, method PaymentMethod, invoice *PaymentInvoiceOpts, mirrorID string) *e.ErrorInfo {
	bot, err := GetBotByMirrorID(mirrorID)
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

func sendInformationalMessage(chatID int64, text string, mirrorID string) {
	bot, err := GetBotByMirrorID(mirrorID)
	if e.IsNonNil(err) {
		return
	}
	_, _ = bot.Send(&tele.Chat{ID: chatID}, text)
}

func sendMirrorPurchaseSuccessMessage(payment *models.Payment, metadata *paymentServiceMetadata) *e.ErrorInfo {
	chatID, err := getPaymentChatID(payment, metadata)
	if e.IsNonNil(err) {
		return err
	}
	mirror, err := models.FindMirrorByID(GetDB(), metadata.Mirror.PendingMirrorID)
	if e.IsNonNil(err) {
		return err
	}
	token, err := mirror.DecryptToken(mirror.Owner)
	if e.IsNonNil(err) {
		return err
	}
	mirrorBot, rawErr := tele.NewBot(tele.Settings{Token: token, Poller: nil})
	if rawErr != nil {
		return e.FromError(rawErr, "failed to initialize mirror bot").WithSeverity(e.Notice)
	}
	if err := setMirrorWebhook(mirrorBot, mirror.Unique); e.IsNonNil(err) {
		return err
	}

	text, entities := buildMirrorSuccessMessage(mirror.BotUsername)
	bot, err := GetBot()
	if e.IsNonNil(err) {
		return err
	}
	_, rawErr = bot.Send(&tele.Chat{ID: chatID}, text, &tele.SendOptions{Entities: entities})
	if rawErr != nil {
		return e.FromError(rawErr, "failed to send mirror purchase success message").WithSeverity(e.Notice)
	}
	return e.Nil()
}

func sendLevelPurchaseSuccessMessage(payment *models.Payment, metadata *paymentServiceMetadata) *e.ErrorInfo {
	chatID, err := getPaymentChatID(payment, metadata)
	if e.IsNonNil(err) {
		return err
	}

	text, entities := buildLevelPurchaseSuccessMessage(metadata.LevelUp.Levels)
	bot, err := GetBot()
	if e.IsNonNil(err) {
		return err
	}

	_, rawErr := bot.Send(&tele.Chat{ID: chatID}, text, &tele.SendOptions{Entities: entities})
	if rawErr != nil {
		return e.FromError(rawErr, "failed to send level purchase success message").WithSeverity(e.Notice)
	}
	return e.Nil()
}

func getPaymentChatID(payment *models.Payment, metadata *paymentServiceMetadata) (int64, *e.ErrorInfo) {
	if metadata.ChatID != 0 {
		return metadata.ChatID, e.Nil()
	}

	user := &models.Telegramuser{ID: payment.ClientID}
	if rawErr := GetDB().Model(user).WherePK().Select(); rawErr != nil {
		return 0, e.FromError(rawErr, "failed to get payment user").WithSeverity(e.Notice)
	}
	return user.GetTgId()
}

func buildLevelPurchaseSuccessMessage(levels int) (string, tele.Entities) {
	text := fmt.Sprintf(
		"Поздравляем с приобретением %d уровня!⬆️\n\nТеперь все, у кого уровень ниже, не смогут увидеть сообщения, которые ты изменяешь и удаляешь через %s, и не будут иметь возможности сохранить медиа с одним просмотром, отправленные тобой",
		levels,
		levelPurchaseBotMention,
	)
	boldText := fmt.Sprintf("Поздравляем с приобретением %d уровня!", levels)
	customEmojiOffset := utils.TgLen(boldText)
	mentionOffset := utils.TgLen(fmt.Sprintf(
		"Поздравляем с приобретением %d уровня!⬆️\n\nТеперь все, у кого уровень ниже, не смогут увидеть сообщения, которые ты изменяешь и удаляешь через ",
		levels,
	))

	return text, tele.Entities{
		{Type: tele.EntityBold, Offset: 0, Length: customEmojiOffset},
		{Type: tele.EntityCustomEmoji, Offset: customEmojiOffset, Length: 2, CustomEmojiID: "5463122435425448565"},
		{Type: tele.EntityMention, Offset: mentionOffset, Length: utils.TgLen(levelPurchaseBotMention)},
	}
}

func buildMirrorSuccessMessage(botUsername string) (string, tele.Entities) {
	botMention := "@" + botUsername
	text := fmt.Sprintf(
		"Зеркало успешно создано!👉\n\nОно функционирует точно так же, как и %s.\nЕсли он не будет использоваться больше месяца, то %s будет отключён от системы",
		levelPurchaseBotMention,
		botMention,
	)
	return text, tele.Entities{
		{Offset: 24, Length: 2, Type: tele.EntityCustomEmoji, CustomEmojiID: "5463392464314315076"},
		{Offset: 66, Length: utils.TgLen(levelPurchaseBotMention), Type: tele.EntityMention},
	}
}

func setMirrorWebhook(bot *tele.Bot, unique string) *e.ErrorInfo {
	if bot == nil {
		return e.NewError("bot is nil", "failed to set mirror webhook").WithSeverity(e.Notice)
	}
	if err := bot.SetWebhook(&tele.Webhook{
		Endpoint:       &tele.WebhookEndpoint{PublicURL: buildMirrorWebhookURL(unique)},
		MaxConnections: 100,
		AllowedUpdates: []string{
			"message",
			"callback_query",
			"shipping_query",
			"pre_checkout_query",
			"business_connection",
			"business_message",
			"edited_business_message",
			"deleted_business_messages",
		},
	}); err != nil {
		return e.FromError(err, "failed to set mirror webhook").WithSeverity(e.Notice)
	}
	return e.Nil()
}

func buildMirrorWebhookURL(unique string) string {
	base := strings.TrimRight(os.Getenv("TELEGRAM_BOT_PUBLIC_URL"), "/")
	return base + "/mirror/" + unique
}
