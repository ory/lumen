package com.example.shop.checkout;

import com.example.shop.payments.ChargeResult;
import com.example.shop.payments.PaymentGateway;

public class CheckoutService {

    private final PaymentGateway payments;
    private final OrderRepository orders;

    public CheckoutService(PaymentGateway payments, OrderRepository orders) {
        this.payments = payments;
        this.orders = orders;
    }

    public Order placeOrder(Cart cart, String customerId) {
        Order order = orders.save(Order.pending(cart, customerId));
        ChargeResult result = payments.charge(customerId, cart.totalCents(), cart.currency(), "order-" + order.id());
        if (!result.isSuccessful()) {
            return orders.save(order.declined(result.declineCode()));
        }
        return orders.save(order.paid(result.chargeId()));
    }
}
