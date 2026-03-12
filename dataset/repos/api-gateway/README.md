# api-gateway

The API Gateway is the single entry point for all external traffic.

## Overview
The API Gateway routes incoming requests to AuthService and PaymentService.
All external clients communicate exclusively through the API Gateway.

## Dependencies
- auth-service: handles authentication requests
- payment-service: handles payment requests

## Routing
/auth/*     → routes to AuthService
/payments/* → routes to PaymentService
