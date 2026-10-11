import { gateway } from '../gateway';

export async function chargeDeposit(guestId: string, amountCents: number): Promise<string> {
  const intent = await gateway.authorize({ customer: guestId, amount: amountCents, capture: true });
  return intent.id;
}

export async function refundDeposit(depositId: string): Promise<void> {
  await gateway.refund({ intent: depositId });
}
