package main

import (
	"errors"
	"net/http"
	"time"
)

const (
	stripeTimeout    = 10 * time.Second
	stripeAPIVersion = "2023-10-16"
)

var ErrStripeUnavailable = errors.New("stripe service unavailable")

type StripeClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewStripeClient(apiKey string) *StripeClient {
	return &StripeClient{
		apiKey:  apiKey,
		baseURL: "https://api.stripe.com/v1",
		client:  &http.Client{Timeout: stripeTimeout},
	}
}

func (s *StripeClient) Charge(amount int, currency string) (string, error) {
	// calls POST /v1/charges on Stripe API
	return "", nil
}

func (s *StripeClient) Refund(txID string) error {
	// calls POST /v1/refunds on Stripe API
	return nil
}
