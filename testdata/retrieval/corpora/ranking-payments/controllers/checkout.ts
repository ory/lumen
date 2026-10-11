import { processPayment } from '../billing/payment';
import { redisIdempotencyStore } from '../infra/redis';

// POST /checkout: reads the Idempotency-Key header and charges the cart total.
export async function checkout(req: Request): Promise<Response> {
  const idempotencyKey = req.headers.get('Idempotency-Key') ?? crypto.randomUUID();
  const cart = await req.json();
  const result = await processPayment(
    { idempotencyKey, amountCents: cart.totalCents, cardToken: cart.cardToken },
    redisIdempotencyStore,
  );
  return Response.json(result, { status: result.status === 'captured' ? 201 : 402 });
}
