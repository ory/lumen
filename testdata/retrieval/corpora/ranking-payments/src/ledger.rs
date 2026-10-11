use std::collections::HashSet;

pub type PaymentId = u64;

/// Records a payment id before processing so a duplicate delivery from the
/// queue is acknowledged without a second capture.
pub fn record_once(seen: &mut HashSet<PaymentId>, id: PaymentId) -> bool {
    seen.insert(id)
}
