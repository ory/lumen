package pricing

// Item 是购物车中的一行。
type Item struct {
	Price float64
	Qty   int
}

// 计算订单总价，包含税费。
func 计算总价(items []Item, 税率 float64) float64 {
	var 小计 float64
	for _, it := range items {
		小计 += it.Price * float64(it.Qty)
	}
	return 小计 * (1 + 税率)
}

// 计算运费，满额包邮。
func 计算运费(小计 float64) float64 {
	if 小计 >= 99 {
		return 0
	}
	return 8
}
