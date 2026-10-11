package inventory

import (
	"context"
	"errors"
)

// ErrInsufficientStock is returned when a SKU cannot cover the requested quantity.
var ErrInsufficientStock = errors.New("insufficient stock")

// Reserve holds qty units of sku until the hold expires.
func Reserve(ctx context.Context, sku string, qty int) error {
	available, err := stockLevel(ctx, sku)
	if err != nil {
		return err
	}
	if available < qty {
		return ErrInsufficientStock
	}
	return holds.put(ctx, sku, qty)
}
