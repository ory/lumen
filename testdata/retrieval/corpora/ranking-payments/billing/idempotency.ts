import { PaymentResult } from './gateway';

export interface StoredAttempt {
  amountCents: number;
  result: PaymentResult;
}

// Key-value store for completed attempts, keyed by client-supplied key.
export interface IdempotencyStore {
  get(key: string): Promise<StoredAttempt | undefined>;
  put(key: string, attempt: StoredAttempt): Promise<void>;
}
