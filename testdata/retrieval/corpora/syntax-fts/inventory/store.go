package inventory

import "context"

// Store is the persistence port used by the inventory service.
type Store interface {
	OnHand(ctx context.Context, sku string) (int, error)
	Hold(ctx context.Context, sku string, qty int) error
	Unhold(ctx context.Context, sku string, qty int) error
}
