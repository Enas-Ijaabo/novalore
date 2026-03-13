package main

import (
	"database/sql"
	"errors"
	"net/http"
	"time"
)

const (
	maxRetries      = 3
	retryDelay      = 2 * time.Second
	minChargeAmount = 50     // cents — minimum charge is $0.50
	maxChargeAmount = 999999 // cents — maximum single charge is $9,999.99
	refundWindowDays = 30
)

var (
	ErrAmountTooLow    = errors.New("charge amount below minimum of $0.50")
	ErrAmountTooHigh   = errors.New("charge amount exceeds maximum of $9,999.99")
	ErrRefundExpired   = errors.New("refund window of 30 days has passed")
	ErrAlreadyRefunded = errors.New("transaction has already been refunded")
)

type PaymentService struct {
	db     *sql.DB
	stripe *StripeClient
}

func (p *PaymentService) Charge(userID string, amount int, currency string) (string, error) {
	if amount < minChargeAmount {
		return "", ErrAmountTooLow
	}
	if amount > maxChargeAmount {
		return "", ErrAmountTooHigh
	}

	var txID string
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		txID, lastErr = p.stripe.Charge(amount, currency)
		if lastErr == nil {
			break
		}
		time.Sleep(retryDelay)
	}
	if lastErr != nil {
		return "", lastErr
	}

	p.db.Exec(
		"INSERT INTO transactions (id, user_id, amount, currency, status, created_at) VALUES ($1, $2, $3, $4, 'completed', $5)",
		txID, userID, amount, currency, time.Now(),
	)
	return txID, nil
}

func (p *PaymentService) Refund(txID string) error {
	var createdAt time.Time
	var refunded bool
	p.db.QueryRow(
		"SELECT created_at, refunded FROM transactions WHERE id = $1", txID,
	).Scan(&createdAt, &refunded)

	if refunded {
		return ErrAlreadyRefunded
	}
	if time.Since(createdAt) > time.Duration(refundWindowDays)*24*time.Hour {
		return ErrRefundExpired
	}

	p.stripe.Refund(txID)
	p.db.Exec("UPDATE transactions SET refunded = true WHERE id = $1", txID)
	return nil
}

func (p *PaymentService) chargeHandler(w http.ResponseWriter, r *http.Request)         {}
func (p *PaymentService) refundHandler(w http.ResponseWriter, r *http.Request)          {}
func (p *PaymentService) getTransactionHandler(w http.ResponseWriter, r *http.Request) {}
