package com.example.shop.payments;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * In-memory gateway for tests. Records every charge and can be told to decline
 * specific customers.
 */
public class FakePaymentGateway implements PaymentGateway {

    private final Map<String, ChargeResult> chargesByIdempotencyKey = new HashMap<>();
    private final Map<String, String> declinedCustomers = new HashMap<>();
    private final List<String> refunds = new ArrayList<>();

    public void declineCustomer(String customerId, String declineCode) {
        declinedCustomers.put(customerId, declineCode);
    }

    @Override
    public ChargeResult charge(String customerId, long amountCents, String currency, String idempotencyKey) {
        return chargesByIdempotencyKey.computeIfAbsent(idempotencyKey, key -> {
            String declineCode = declinedCustomers.get(customerId);
            if (declineCode != null) {
                return ChargeResult.declined(declineCode);
            }
            return ChargeResult.succeeded("ch_fake_" + UUID.randomUUID());
        });
    }

    @Override
    public void refund(String chargeId, long amountCents) {
        refunds.add(chargeId + ":" + amountCents);
    }

    public int chargeCount() {
        return chargesByIdempotencyKey.size();
    }

    public List<String> refunds() {
        return List.copyOf(refunds);
    }
}
