export function convertCurrency(amountCents, from, to, rates) {
  if (from === to) {
    return amountCents;
  }
  return Math.round((amountCents / rates[from]) * rates[to]);
}
