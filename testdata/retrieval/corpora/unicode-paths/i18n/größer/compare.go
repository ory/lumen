package groesser

// CompareGarmentWidths orders two garments by chest width in millimetres.
func CompareGarmentWidths(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
