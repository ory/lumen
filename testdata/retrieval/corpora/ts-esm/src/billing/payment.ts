import { reserveSlot as claim } from './reservation.js';

export interface PaymentRequest {
  id: string;
  amountCents: number;
}

export function processPayment(request: PaymentRequest): 'accepted' | 'duplicate' {
  if (!claim(request.id)) {
    return 'duplicate';
  }
  return 'accepted';
}
