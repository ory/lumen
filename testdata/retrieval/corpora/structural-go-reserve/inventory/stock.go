package inventory

import "context"

type holdTable struct{ bySKU map[string]int }

var holds = &holdTable{bySKU: map[string]int{}}

func (h *holdTable) put(_ context.Context, sku string, qty int) error {
	h.bySKU[sku] += qty
	return nil
}

func stockLevel(_ context.Context, sku string) (int, error) {
	return 100 - holds.bySKU[sku], nil
}
