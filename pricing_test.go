package paymentservice

import "testing"

func TestCalculateLevelUpPrice(t *testing.T) {
	price, err := calculatePrice(PaymentTypeLevelUp, &PaymentOpts{LevelUp: &LevelUpOpts{Levels: 3}})
	if !err.IsNil() {
		t.Fatalf("unexpected error: %s", err.JSON())
	}
	if price != 3 {
		t.Fatalf("expected 3 stars, got %d", price)
	}
}

func TestCalculateExportChatIsNotEnabledYet(t *testing.T) {
	price, err := calculatePrice(PaymentTypeExportChat, &PaymentOpts{ExportChat: &ExportChatOpts{Messages: 6}})
	if err.IsNil() {
		t.Fatal("expected export chat to be unsupported")
	}
	if price != 0 {
		t.Fatalf("expected zero price for unsupported payment, got %d", price)
	}
}

func TestCalculateMirrorPrice(t *testing.T) {
	price, err := calculatePrice(PaymentTypeMirror, &PaymentOpts{Mirror: &MirrorOpts{PendingMirrorID: 42}})
	if !err.IsNil() {
		t.Fatalf("unexpected error: %s", err.JSON())
	}
	if price != 1 {
		t.Fatalf("expected 1 star, got %d", price)
	}
}

func TestCalculatePriceRejectsInvalidLevelCount(t *testing.T) {
	_, err := calculatePrice(PaymentTypeLevelUp, &PaymentOpts{LevelUp: &LevelUpOpts{Levels: 0}})
	if err.IsNil() {
		t.Fatal("expected error for zero levels")
	}
}

func TestConvertStarsAmountUsesRate(t *testing.T) {
	amount := convertStarsAmount(10, PaymentMethod{StarsRate: 1.5})
	if amount != 15 {
		t.Fatalf("expected 15, got %d", amount)
	}
}
