# Storefront payments

## Usage

Checkout calls `processPayment` with the cart total. Clients should send an
`Idempotency-Key` header; see the design doc for how idempotency works.

## Development

Run `npm test` to execute the payment and checkout suites.
