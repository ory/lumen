using System;
using System.Linq;

namespace Shop.Billing;

public partial class InvoiceService
{
    // VAT is charged per line using the rate of the buyer's country; exempt lines are skipped.
    public partial decimal CalculateTax(Invoice invoice)
    {
        var rate = _rates.VatRateFor(invoice.BuyerCountry);
        var taxable = invoice.Lines.Where(line => !line.TaxExempt).Sum(line => line.Amount);
        return Math.Round(taxable * rate, 2, MidpointRounding.ToEven);
    }
}
