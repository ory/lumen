package com.example.shop.payments;

/**
 * Abstraction over the payment provider used by checkout.
 */
public interface PaymentGateway {

    /**
     * Charges the customer's default payment method.
     * @param idempotencyKey key that makes retries of the same charge safe
     */
    ChargeResult charge(String customerId, long amountCents, String currency, String idempotencyKey);

    /**
     * Refunds part or all of a previous charge.
     */
    void refund(String chargeId, long amountCents);
}
