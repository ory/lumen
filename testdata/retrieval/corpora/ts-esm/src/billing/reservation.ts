import { ledger } from './ledger.js';

export function reserveSlot(requestId: string): boolean {
  return ledger.insertIfAbsent(`slot:${requestId}`);
}

export function slotKey(requestId: string): string {
  return `slot:${requestId}`;
}
