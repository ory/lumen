package inventory

import (
	"context"
	"errors"
)

// ErrInsufficientStock is returned when fewer units are on hand than requested.
var ErrInsufficientStock = errors.New("inventory: insufficient stock")

// Reserve holds qty units of sku for an order until the hold expires.
func Reserve(ctx context.Context, store Store, sku string, qty int) error {
	onHand, err := store.OnHand(ctx, sku)
	if err != nil {
		return err
	}
	if onHand < qty {
		return ErrInsufficientStock
	}
	return store.Hold(ctx, sku, qty)
}

// Release returns held units to the available pool.
func Release(ctx context.Context, store Store, sku string, qty int) error {
	return store.Unhold(ctx, sku, qty)
}
