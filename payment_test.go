package paymentservice

import (
	"strings"
	"testing"

	models "github.com/ChatDetectiveORG/shared/postgresModels"
	tele "gopkg.in/telebot.v4"
)

func TestPreCheckoutActionForStatusIsIdempotent(t *testing.T) {
	cases := []struct {
		status string
		want   preCheckoutAction
	}{
		{models.PaymentStatusInvoiceSent, preCheckoutActionGrant},
		{models.PaymentStatusPreCheckoutReceived, preCheckoutActionGrant},
		{models.PaymentStatusApproved, preCheckoutActionReApprove},
		{models.PaymentStatusCancelled, preCheckoutActionReCancel},
		{models.PaymentStatusTimedOut, preCheckoutActionReCancel},
	}
	for _, c := range cases {
		if got := preCheckoutActionForStatus(c.status); got != c.want {
			t.Fatalf("status %q: expected action %d, got %d", c.status, c.want, got)
		}
	}
}

func TestValidateInvoiceRequiresTexts(t *testing.T) {
	err := validateInvoice(&PaymentInvoiceOpts{
		Title:       "Поднятие уровня",
		Description: "Оплата поднятия уровня",
		PriceLabel:  "Уровни",
	})
	if !err.IsNil() {
		t.Fatalf("unexpected error: %s", err.JSON())
	}

	err = validateInvoice(&PaymentInvoiceOpts{})
	if err.IsNil() {
		t.Fatal("expected error for empty invoice texts")
	}
}

func TestValidateRecipientRequiresIDs(t *testing.T) {
	err := validateRecipient(&PaymentRecipientOpts{TelegramUserID: 1, ChatID: 2})
	if !err.IsNil() {
		t.Fatalf("unexpected error: %s", err.JSON())
	}

	err = validateRecipient(&PaymentRecipientOpts{})
	if err.IsNil() {
		t.Fatal("expected error for empty recipient ids")
	}
}

func TestMarshalPaymentMetadataStoresLevelCount(t *testing.T) {
	metadata, err := marshalPaymentMetadata(PaymentTypeLevelUp, &PaymentOpts{
		Recipient: &PaymentRecipientOpts{TelegramUserID: 1, ChatID: 777},
		LevelUp:   &LevelUpOpts{Levels: 5},
	})
	if !err.IsNil() {
		t.Fatalf("unexpected error: %s", err.JSON())
	}
	if !strings.Contains(metadata, `"levels":5`) {
		t.Fatalf("expected level count in metadata, got %s", metadata)
	}
	if !strings.Contains(metadata, `"chat_id":777`) {
		t.Fatalf("expected chat id in metadata, got %s", metadata)
	}
}

func TestBuildLevelPurchaseSuccessMessageEntities(t *testing.T) {
	text, entities := buildLevelPurchaseSuccessMessage(1)

	if !strings.Contains(text, "Поздравляем с приобретением 1 уровня!") {
		t.Fatalf("unexpected text: %s", text)
	}
	if len(entities) != 3 {
		t.Fatalf("expected 3 entities, got %d", len(entities))
	}

	if entities[0].Type != tele.EntityBold || entities[0].Offset != 0 || entities[0].Length != 37 {
		t.Fatalf("unexpected bold entity: %+v", entities[0])
	}
	if entities[1].Type != tele.EntityCustomEmoji || entities[1].Offset != 37 || entities[1].Length != 2 || entities[1].CustomEmojiID != "5463122435425448565" {
		t.Fatalf("unexpected custom emoji entity: %+v", entities[1])
	}
	if entities[2].Type != tele.EntityMention || entities[2].Offset != 141 || entities[2].Length != 25 {
		t.Fatalf("unexpected mention entity: %+v", entities[2])
	}
}
