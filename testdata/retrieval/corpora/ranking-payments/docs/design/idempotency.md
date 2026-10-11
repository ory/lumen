# Idempotent payments

## Motivation

Mobile clients retry on flaky networks, so the same checkout may arrive more
than once. We want every retry to observe the first outcome.

## Approach

`processPayment` looks up the idempotency key before calling the gateway and
stores the result afterwards. Keys expire after 24 hours in Redis.
