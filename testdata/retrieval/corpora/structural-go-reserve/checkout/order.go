package checkout

import (
	"context"
	"fmt"

	inv "example.com/shop/inventory"
)

// LineItem is one SKU on an order.
type LineItem struct {
	SKU string
	Qty int
}

// PlaceOrder holds stock for every line item before the order is charged.
func PlaceOrder(ctx context.Context, items []LineItem) error {
	for _, item := range items {
		if err := inv.Reserve(ctx, item.SKU, item.Qty); err != nil {
			return fmt.Errorf("reserve %s: %w", item.SKU, err)
		}
	}
	return nil
}
