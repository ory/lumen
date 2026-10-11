package warehouse

import "context"

// Reserve books a loading dock for an inbound shipment.
func Reserve(ctx context.Context, dock string, qty int) error {
	return docks.book(ctx, dock, qty)
}

type dockBook struct{ booked map[string]int }

var docks = &dockBook{booked: map[string]int{}}

func (d *dockBook) book(_ context.Context, dock string, pallets int) error {
	d.booked[dock] += pallets
	return nil
}
