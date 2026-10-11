// Receipt formatting for 58mm thermal printers.
import type { Line } from './invoice_lines';

/** Pads each receipt line to the printer width. */
export function formatReceiptLines(lines: Line[], width = 32): string[] {
  return lines.map((l) => {
    const left = `${l.qty}x ${l.description}`;
    return left.padEnd(width - l.amount.length) + l.amount;
  });
}
