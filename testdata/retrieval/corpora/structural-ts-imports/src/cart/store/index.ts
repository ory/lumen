import type { Cart } from '../cart';
import { redis } from '../../redis';

export async function saveCart(cart: Cart): Promise<void> {
  await redis.set(`cart:${cart.id}`, JSON.stringify(cart));
}
