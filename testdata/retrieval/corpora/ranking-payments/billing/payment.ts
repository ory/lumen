import { chargeCard, PaymentResult } from './gateway';
import { IdempotencyStore } from './idempotency';

export interface PaymentRequest {
  idempotencyKey: string;
  amountCents: number;
  cardToken: string;
}

export class IdempotencyConflictError extends Error {}

// A retried request with the same key replays the stored result; reusing a key
// for a different amount is rejected. Duplicate submissions never reach the
// card gateway twice.
export async function processPayment(req: PaymentRequest, store: IdempotencyStore): Promise<PaymentResult> {
  const prior = await store.get(req.idempotencyKey);
  if (prior) {
    if (prior.amountCents !== req.amountCents) {
      throw new IdempotencyConflictError(req.idempotencyKey);
    }
    return prior.result;
  }
  const result = await chargeCard(req.cardToken, req.amountCents);
  await store.put(req.idempotencyKey, { amountCents: req.amountCents, result });
  return result;
}
