pub struct LineItem {
    pub label: String,
    pub cents: i64,
}

/// Renders the invoice with its line-item charges and the grand total.
pub fn render_invoice(number: &str, charges: &[LineItem]) -> String {
    let mut out = format!("Invoice {number}\n");
    for c in charges {
        out.push_str(&format!("{:<30}{:>10}\n", c.label, c.cents));
    }
    let total: i64 = charges.iter().map(|c| c.cents).sum();
    out.push_str(&format!("{:<30}{:>10}\n", "Total", total));
    out
}
