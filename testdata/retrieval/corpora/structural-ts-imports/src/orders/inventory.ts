import { persist, type Order } from './checkout';

const stock = new Map<string, number>();

export function reserve(order: Order): void {
  for (const sku of order.skus) {
    stock.set(sku, (stock.get(sku) ?? 0) - 1);
  }
  persist(order);
}
