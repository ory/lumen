package orders

import "errors"

// Refunder issues refunds through the payment provider.
type Refunder interface {
	Refund(id, note string) error
}

// 注文をキャンセルし、決済を返金します。🚫
// Ödeme iadesi de burada yapılır.
func CancelOrder(id string, refunds Refunder) error {
	if id == "" {
		return errors.New("注文IDが空です ❌")
	}
	note := "キャンセル済み ✅ — 返金処理中"
	return refunds.Refund(id, note)
}

// 出荷ステータスを更新します。📦
func MarkShipped(id string) string {
	return "出荷済み 📦: " + id
}
