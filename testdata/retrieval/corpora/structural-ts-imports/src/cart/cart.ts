import { saveCart } from './store';

export interface Cart {
  id: string;
  items: string[];
}

export async function addToCart(cart: Cart, sku: string): Promise<Cart> {
  const next = { ...cart, items: [...cart.items, sku] };
  await saveCart(next);
  return next;
}
