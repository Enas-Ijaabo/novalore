# auth-service

AuthService handles authentication for the platform.

## Overview
AuthService validates user credentials and issues JWT tokens.
JWT tokens expire after 24 hours.

## Dependencies
- PostgreSQL: user storage
- JWT: token generation and validation

## Endpoints
POST /auth/login    - authenticate user, returns JWT token
POST /auth/refresh  - refresh an expiring JWT token
POST /auth/logout   - invalidate a JWT token
