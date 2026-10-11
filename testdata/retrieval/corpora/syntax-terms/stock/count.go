package stock

// CountWarehouseUnits totals units across all bins of a warehouse.
func CountWarehouseUnits(bins []int) int {
	total := 0
	for _, b := range bins {
		total += b
	}
	return total
}
