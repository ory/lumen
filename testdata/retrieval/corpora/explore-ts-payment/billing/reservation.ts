import { ledger } from './ledger';

// Records the key so that concurrent or retried requests observe it.
export async function reserveSlot(key: string): Promise<boolean> {
  return ledger.insertIfAbsent(`payment:${key}`, Date.now());
}

export async function releaseSlot(key: string): Promise<void> {
  await ledger.remove(`payment:${key}`);
}
