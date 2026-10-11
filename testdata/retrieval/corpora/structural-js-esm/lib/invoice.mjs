import { formatMoney } from './format.mjs';
import { convertCurrency } from './currency';

// Renders the invoice total in the customer's display currency.
export function renderInvoiceTotal(invoice, displayCurrency, rates) {
  const total = invoice.lines.reduce((sum, line) => sum + line.amountCents, 0);
  const converted = convertCurrency(total, invoice.currency, displayCurrency, rates);
  return formatMoney(converted, displayCurrency);
}
