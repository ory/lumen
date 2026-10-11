export function reserveSlot(key: string) { return ledger.insertIfAbsent(key); }
