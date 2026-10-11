using System;

namespace Shop.Shipping;

public class InvoiceService
{
    private readonly ICustomsTariffs _tariffs;

    public InvoiceService(ICustomsTariffs tariffs)
    {
        _tariffs = tariffs;
    }

    // Import duty for cross-border parcels; domestic shipments are duty free.
    public decimal CalculateTax(Shipment shipment)
    {
        if (shipment.OriginCountry == shipment.DestinationCountry)
        {
            return 0m;
        }
        var duty = _tariffs.DutyRate(shipment.HsCode, shipment.DestinationCountry);
        return Math.Round(shipment.DeclaredValue * duty, 2);
    }
}

public sealed record Shipment(string HsCode, string OriginCountry, string DestinationCountry, decimal DeclaredValue);

public interface ICustomsTariffs
{
    decimal DutyRate(string hsCode, string country);
}
