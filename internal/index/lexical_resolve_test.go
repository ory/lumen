// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package index

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// relatedOf returns "file:symbol" for every related snippet of query.
func relatedOf(t *testing.T, idx *Indexer, root, query string) []string {
	t.Helper()
	results, err := idx.SearchLexical(context.Background(), root, query, 3, "", true)
	if err != nil {
		t.Fatal(err)
	}
	var related []string
	for _, r := range results {
		if r.RelatedTo != "" {
			related = append(related, filepath.ToSlash(r.FilePath)+":"+r.Symbol)
		}
	}
	return related
}

func TestLexicalRelatedResolution(t *testing.T) {
	for _, tt := range []struct {
		name  string
		files map[string]string
		query string
		want  []string
	}{
		{
			name: "js specifier maps to ts source",
			files: map[string]string{
				"pay/payment.ts":     "import { reserveSlot as claim } from './reservation.js';\nexport function processPayment(id: string) { return claim(id); }\n",
				"pay/reservation.ts": "export function reserveSlot(key: string) { return key; }\n",
			},
			query: "processPayment",
			want:  []string{"pay/reservation.ts:reserveSlot"},
		},
		{
			name: "mjs specifier maps to mts source",
			files: map[string]string{
				"jobs/report.mts": "import { formatRows } from './format.mjs';\nexport function buildMonthlyReport() { return formatRows([]); }\n",
				"jobs/format.mts": "export function formatRows(rows: string[]) { return rows; }\n",
			},
			query: "buildMonthlyReport",
			want:  []string{"jobs/format.mts:formatRows"},
		},
		{
			name: "js and ts siblings are ambiguous",
			files: map[string]string{
				"pay/payment.ts":     "import { reserveSlot } from './reservation.js';\nexport function processPayment(id: string) { return reserveSlot(id); }\n",
				"pay/reservation.ts": "export function reserveSlot(key: string) { return key; }\n",
				"pay/reservation.js": "export function reserveSlot(key) { return key; }\n",
			},
			query: "processPayment",
		},
		{
			name: "export clause",
			files: map[string]string{
				"refunds/refund.ts": "import { releaseHold } from './holds';\nexport function refundPayment(id: string) { return releaseHold(id); }\n",
				"refunds/holds.ts":  "function releaseHold(id: string) { return id; }\nfunction holdCount() { return 0; }\nexport { releaseHold, holdCount };\n",
			},
			query: "refundPayment",
			want:  []string{"refunds/holds.ts:releaseHold"},
		},
		{
			name: "default import of direct default export",
			files: map[string]string{
				"billing/issue.ts":     "import render from './statement';\nexport function issueInvoice(id: string) { return render(id); }\n",
				"billing/statement.ts": "export default function formatStatement(id: string) { return id; }\nexport function unrelatedHelper() {}\n",
			},
			query: "issueInvoice",
			want:  []string{"billing/statement.ts:formatStatement"},
		},
		{
			name: "unexported import target",
			files: map[string]string{
				"money/a.ts":     "import { roundCents } from './money';\nexport function total(n: number) { return roundCents(n); }\n",
				"money/money.ts": "function roundCents(n: number) { return n; }\n",
			},
			query: "total",
		},
		{
			name: "cycle expands one hop only",
			files: map[string]string{
				"orders/checkout.ts":  "import { reserve } from './inventory';\nexport function checkout(id: string) { return reserve(id); }\nexport function persist(id: string) { return id; }\n",
				"orders/inventory.ts": "import { persist } from './checkout';\nexport function reserve(id: string) { return persist(id); }\n",
			},
			query: "checkout",
			want:  []string{"orders/inventory.ts:reserve"},
		},
		{
			name: "destructured local shadows imported alias",
			files: map[string]string{
				"refunds/refund.ts": "import { reverseCharge as claim } from './ledger';\nexport function issueRefund(id: string, deps: any) { const { claim } = deps; return claim(id); }\n",
				"refunds/ledger.ts": "export function reverseCharge(id: string) { return id; }\n",
			},
			query: "issueRefund",
		},
		{
			name: "go module import excludes tests and other packages",
			files: map[string]string{
				"go.mod":                    "module example.com/shop\n\ngo 1.22\n",
				"orders/place.go":           "package orders\n\nimport inv \"example.com/shop/inventory\"\n\nfunc PlaceOrder(id string) error { return inv.Reserve(id) }\n",
				"inventory/reserve.go":      "package inventory\n\nfunc Reserve(id string) error { return nil }\n",
				"inventory/reserve_test.go": "package inventory_test\n\nfunc Reserve(id string) error { return nil }\n",
				"warehouse/reserve.go":      "package warehouse\n\nfunc Reserve(id string) error { return nil }\n",
			},
			query: "PlaceOrder",
			want:  []string{"inventory/reserve.go:Reserve"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			idx, root := lexicalFixture(t, tt.files)
			if got := relatedOf(t, idx, root, tt.query); !slices.Equal(got, tt.want) {
				t.Fatalf("related = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLexicalAmbiguousModuleResolvesAfterRemoval(t *testing.T) {
	idx, root := lexicalFixture(t, map[string]string{
		"basket/basket.ts":      "import { persistItems } from './store';\nexport function addToCart(id: string) { return persistItems(id); }\n",
		"basket/store.ts":       "export function persistItems(id: string) { return id; }\n",
		"basket/store/index.ts": "export function persistItems(id: string) { return id + '!'; }\n",
	})
	if got := relatedOf(t, idx, root, "addToCart"); len(got) != 0 {
		t.Fatalf("ambiguous ./store expanded to %v", got)
	}
	if err := os.Remove(filepath.Join(root, "basket", "store", "index.ts")); err != nil {
		t.Fatal(err)
	}
	if _, err := idx.Index(context.Background(), root, false, nil); err != nil {
		t.Fatal(err)
	}
	if got := relatedOf(t, idx, root, "addToCart"); !slices.Equal(got, []string{"basket/store.ts:persistItems"}) {
		t.Fatalf("after removal related = %v", got)
	}
}

// sizedFunction returns a one-line TypeScript function of exactly n UTF-8
// bytes, padded with two-byte runes so the budget is measured in bytes, not
// runes.
func sizedFunction(name string, n int) string {
	head, tail := "export function "+name+"() { return '", "'; }"
	pad := n - len(head) - len(tail)
	return head + strings.Repeat("é", pad/2) + strings.Repeat("x", pad%2) + tail
}

func TestLexicalRelatedByteBudget(t *testing.T) {
	for _, tt := range []struct {
		bigBytes int
		want     []string
	}{
		// The budget is 6000 UTF-8 bytes across complete snippets.
		{5999, []string{"lib/big.ts:bigHelper"}},
		{6000, []string{"lib/big.ts:bigHelper"}},
		{6001, []string{"lib/small.ts:smallHelper"}},
	} {
		idx, root := lexicalFixture(t, map[string]string{
			"lib/entry.ts": "import { bigHelper } from './big';\nimport { smallHelper } from './small';\nexport function entryPoint() { bigHelper(); smallHelper(); }\n",
			"lib/big.ts":   sizedFunction("bigHelper", tt.bigBytes) + "\n",
			"lib/small.ts": "export function smallHelper() { return 1; }\n",
		})
		if got := relatedOf(t, idx, root, "entryPoint"); !slices.Equal(got, tt.want) {
			t.Errorf("big=%d bytes: related = %v, want %v", tt.bigBytes, got, tt.want)
		}
	}
}
