// Voucher codes issued by the previous campaign system.
export function applyLegacyVoucher(code: string, cents: number): number {
  return code.startsWith('LEG') ? Math.round(cents / 2) : cents;
}
