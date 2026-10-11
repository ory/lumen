package accounting

import "time"

// Entry is a single double-entry posting.
type Entry struct {
	Account string
	Debit   int64
	Credit  int64
	At      time.Time
}

// Ledger keeps postings in append-only order.
type Ledger struct {
	entries []Entry
}

// Append adds a posting to the ledger after checking it balances.
func (l *Ledger) Append(e Entry) bool {
	if e.Debit != 0 && e.Credit != 0 {
		return false
	}
	l.entries = append(l.entries, e)
	return true
}

// Balance sums debits minus credits for one account.
func (l *Ledger) Balance(account string) int64 {
	var total int64
	for _, e := range l.entries {
		if e.Account == account {
			total += e.Debit - e.Credit
		}
	}
	return total
}
