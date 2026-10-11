import { reserveSlot as claim, releaseSlot } from './reservation';
import { chargeCard, refundCharge } from './gateway';

export interface PaymentRequest {
  idempotencyKey: string;
  customerId: string;
  amountCents: number;
}

// Charges the customer once per idempotency key. A second request with the
// same key is acknowledged without charging the card again.
export async function processPayment(req: PaymentRequest): Promise<string> {
  if (!(await claim(req.idempotencyKey))) {
    return 'duplicate';
  }
  try {
    return await chargeCard(req.customerId, req.amountCents);
  } catch (err) {
    await releaseSlot(req.idempotencyKey);
    throw err;
  }
}

export async function refundPayment(chargeId: string, amountCents: number): Promise<void> {
  await refundCharge(chargeId, amountCents);
}
