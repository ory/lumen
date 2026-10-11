import { releaseHold } from './ledger';

export function refundPayment(paymentId: string, reason: string): string {
  releaseHold(paymentId);
  return `refunded ${paymentId}: ${reason}`;
}
