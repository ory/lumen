using System;
using System.Collections.Generic;
using System.Linq;

namespace Shop.Billing;

public sealed record InvoiceLine(string Sku, decimal Amount, bool TaxExempt);

public sealed record Invoice
{
    public Guid Id { get; init; } = Guid.NewGuid();
    public string BuyerCountry { get; init; } = "DE";
    public IReadOnlyList<InvoiceLine> Lines { get; init; } = Array.Empty<InvoiceLine>();
    public decimal Tax { get; init; }
    public DateTimeOffset? IssuedAt { get; init; }

    public decimal Total => Lines.Sum(line => line.Amount) + Tax;
}

public interface ITaxRateProvider
{
    decimal VatRateFor(string country);
}
