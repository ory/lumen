import { reserve } from './inventory';

export interface Order {
  id: string;
  skus: string[];
}

export function persist(order: Order): void {
  orderTable.set(order.id, order);
}

const orderTable = new Map<string, Order>();

export function checkout(order: Order): Order {
  reserve(order);
  return order;
}
