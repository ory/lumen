import { reserveSlot as claim } from './reservation';
// Prevent duplicate payment processing.
export function processPayment(id: string) { return claim(id); }
