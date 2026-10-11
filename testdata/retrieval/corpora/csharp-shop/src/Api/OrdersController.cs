using System;
using System.Threading;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using Shop.Orders;

namespace Shop.Api
{
    [ApiController]
    [Route("api")]
    public class OrdersController : ControllerBase
    {
        private readonly IOrderService _orders;

        public OrdersController(IOrderService orders) => _orders = orders;

        [HttpPost("orders")]
        [Authorize(Policy = "orders:write")]
        public async Task<IActionResult> CreateOrder([FromBody] CreateOrderRequest request, CancellationToken ct)
        {
            var summary = await _orders.PlaceAsync(request, ct);
            return CreatedAtAction(nameof(GetOrder), new { id = summary.OrderId }, summary);
        }

        [HttpGet("orders/{id:guid}")]
        [Authorize]
        public async Task<IActionResult> GetOrder(Guid id, CancellationToken ct)
        {
            var summary = await _orders.FindAsync(id, ct);
            return summary is null ? NotFound() : Ok(summary);
        }
    }
}
