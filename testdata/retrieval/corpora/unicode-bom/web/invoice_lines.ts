/** Formats each invoice line as "qty × description — amount". */
export function formatInvoiceLines(lines: Line[]): string[] {
  return lines.map((l) => `${l.qty} × ${l.description} — ${l.amount}`);
}

export interface Line {
  qty: number;
  description: string;
  amount: string;
}

// Totals the printable invoice in minor units.
export function sumInvoiceLines(lines: { cents: number }[]): number {
  return lines.reduce((acc, l) => acc + l.cents, 0);
}
