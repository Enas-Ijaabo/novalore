# payment-service

PaymentService handles all payment processing for the platform.

## Overview
PaymentService processes payments using the Stripe API.
Transaction records are stored in PostgreSQL.

## Dependencies
- Stripe: payment processing
- PostgreSQL: transaction storage

## Endpoints
POST /payments/charge   - charge a payment method via Stripe
GET  /payments/:id      - retrieve a transaction record
