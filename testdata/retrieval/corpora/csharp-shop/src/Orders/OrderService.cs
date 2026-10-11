using System;
using System.Threading;
using System.Threading.Tasks;
using Shop.Data;

namespace Shop.Orders;

public interface IOrderService
{
    Task<OrderSummary> PlaceAsync(CreateOrderRequest request, CancellationToken ct);
    Task<OrderSummary?> FindAsync(Guid id, CancellationToken ct);
}

public sealed class Order : Entity
{
    public int ItemCount { get; set; }
    public decimal Total { get; set; }
}

public sealed class OrderService : IOrderService
{
    private readonly IRepository<Order> _repository;

    public OrderService(IRepository<Order> repository) => _repository = repository;

    public async Task<OrderSummary> PlaceAsync(CreateOrderRequest request, CancellationToken ct)
    {
        var order = new Order { ItemCount = request.Skus.Length, Total = request.Skus.Length * 10m };
        await _repository.AddAsync(order, ct);
        return new OrderSummary(order.Id, order.ItemCount, order.Total);
    }

    public async Task<OrderSummary?> FindAsync(Guid id, CancellationToken ct)
    {
        var order = await _repository.FindAsync(id, ct);
        return order is null ? null : new OrderSummary(order.Id, order.ItemCount, order.Total);
    }
}
