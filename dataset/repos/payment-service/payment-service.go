package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const stripeAPI = "https://api.stripe.com/v1/charges"

type StripeClient struct {
	apiKey string
}

func (s *StripeClient) charge(amount int64, currency string) (string, error) {
	body := strings.NewReader(fmt.Sprintf("amount=%d&currency=%s", amount, currency))
	req, _ := http.NewRequest("POST", stripeAPI, body)
	req.SetBasicAuth(s.apiKey, "")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.ID, nil
}

type PaymentService struct {
	db     *sql.DB
	stripe *StripeClient
}

func NewPaymentService(postgresURL, stripeKey string) *PaymentService {
	db, _ := sql.Open("postgres", postgresURL)
	return &PaymentService{db: db, stripe: &StripeClient{apiKey: stripeKey}}
}

func (p *PaymentService) chargeHandler(w http.ResponseWriter, r *http.Request) {
	chargeID, err := p.stripe.charge(1000, "usd")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.db.Exec("INSERT INTO transactions(stripe_charge_id, status) VALUES($1,'success')", chargeID)
	json.NewEncoder(w).Encode(map[string]string{"charge_id": chargeID})
}

func (p *PaymentService) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/payments/charge", p.chargeHandler)
	return mux
}

func main() {
	svc := NewPaymentService("postgres://localhost/paymentdb", "sk_live_xxx")
	http.ListenAndServe(":8082", svc.routes())
}
