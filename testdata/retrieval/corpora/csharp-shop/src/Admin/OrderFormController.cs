using Microsoft.AspNetCore.Mvc;

namespace Shop.Admin;

public class OrderFormController : Controller
{
    [HttpGet("admin/orders/new")]
    public IActionResult CreateOrder()
    {
        return View("CreateOrder", new OrderFormModel());
    }
}

public class OrderFormModel
{
    public string? CustomerEmail { get; set; }
}
