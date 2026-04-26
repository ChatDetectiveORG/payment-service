package paymentservice

import (
	"testing"
	"time"
)

func TestPaymentDeadlineUsesDefaultWhenEmpty(t *testing.T) {
	before := time.Now().Add(defaultPaymentWait - time.Second)
	deadline := paymentDeadline(nil)
	after := time.Now().Add(defaultPaymentWait + time.Second)

	if deadline.Before(before) || deadline.After(after) {
		t.Fatalf("deadline %s is outside expected default window", deadline)
	}
}

func TestPaymentDeadlineUsesProvidedDeadline(t *testing.T) {
	expected := time.Now().Add(30 * time.Second)
	actual := paymentDeadline(&expected)

	if !actual.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, actual)
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
