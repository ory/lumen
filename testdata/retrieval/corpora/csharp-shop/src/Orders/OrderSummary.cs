using System;

namespace Shop.Orders;

public sealed record OrderSummary(Guid OrderId, int ItemCount, decimal Total);

public sealed record CreateOrderRequest(string CustomerEmail, string[] Skus);
