package pricing

// berechneGesamtsumme summiert alle Posten ohne Rabatte.
func berechneGesamtsumme(posten []Item) float64 {
	var summe float64
	for _, p := range posten {
		summe += p.Price * float64(p.Qty)
	}
	return summe
}

// berechneVersandkosten liefert die Versandkosten nach Gewicht.
func berechneVersandkosten(gewichtKg float64) float64 {
	if gewichtKg <= 2 {
		return 4.9
	}
	return 4.9 + (gewichtKg-2)*1.5
}
