export function formatMoney(amountCents, currency) {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(amountCents / 100);
}
