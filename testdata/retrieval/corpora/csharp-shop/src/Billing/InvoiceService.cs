using System;
using System.Collections.Generic;
using System.Linq;

namespace Shop.Billing
{
    public partial class InvoiceService
    {
        private readonly ITaxRateProvider _rates;

        public InvoiceService(ITaxRateProvider rates)
        {
            _rates = rates;
        }

        public Invoice Finalize(Invoice invoice)
        {
            var tax = CalculateTax(invoice);
            return invoice with { Tax = tax, IssuedAt = DateTimeOffset.UtcNow };
        }

        public partial decimal CalculateTax(Invoice invoice);
    }
}
