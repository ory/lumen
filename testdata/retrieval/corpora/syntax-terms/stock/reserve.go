package stock

import "errors"

var errOutOfStock = errors.New("stock: out of stock")

// ReserveInventoryItem decrements available units and records a hold.
func ReserveInventoryItem(levels map[string]int, holds map[string]int, sku string, qty int) error {
	if levels[sku] < qty {
		return errOutOfStock
	}
	levels[sku] -= qty
	holds[sku] += qty
	return nil
}
