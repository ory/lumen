import Stripe from 'stripe';

const stripe = new Stripe(process.env.STRIPE_KEY ?? '');

export async function chargeCard(customerId: string, amountCents: number): Promise<string> {
  const charge = await stripe.paymentIntents.create({ customer: customerId, amount: amountCents, currency: 'usd', confirm: true });
  return charge.id;
}

export async function refundCharge(chargeId: string, amountCents: number): Promise<void> {
  await stripe.refunds.create({ payment_intent: chargeId, amount: amountCents });
}
