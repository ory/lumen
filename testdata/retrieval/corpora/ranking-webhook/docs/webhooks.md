# Webhooks

## Delivery

Each event is delivered to the subscriber endpoint. Failed deliveries are
retried by the delivery worker; after the final retry the event is moved to
the dead-letter queue and can be replayed from the dashboard.
