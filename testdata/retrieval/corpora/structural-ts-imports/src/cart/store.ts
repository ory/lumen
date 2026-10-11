import type { Cart } from './cart';

const carts = new Map<string, Cart>();

export async function saveCart(cart: Cart): Promise<void> {
  carts.set(cart.id, cart);
}
