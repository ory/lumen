package inventory_test

import (
	"context"
	"testing"

	"example.com/shop/inventory"
)

// Reserve is a test helper that fails the test when the hold cannot be placed.
func Reserve(t *testing.T, sku string, qty int) {
	t.Helper()
	if err := inventory.Reserve(context.Background(), sku, qty); err != nil {
		t.Fatalf("reserve %s: %v", sku, err)
	}
}

func TestReserveRejectsOversell(t *testing.T) {
	Reserve(t, "sku-1", 90)
	if err := inventory.Reserve(context.Background(), "sku-1", 20); err == nil {
		t.Fatal("expected insufficient stock")
	}
}
