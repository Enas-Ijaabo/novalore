package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type AuthService struct {
	db          *sql.DB
	jwtSecret   string
	tokenExpiry time.Duration
}

func NewAuthService(postgresURL, jwtSecret string) *AuthService {
	db, _ := sql.Open("postgres", postgresURL)
	return &AuthService{db: db, jwtSecret: jwtSecret, tokenExpiry: 24 * time.Hour}
}

func (s *AuthService) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", s.login)
	mux.HandleFunc("/auth/refresh", s.refresh)
	mux.HandleFunc("/auth/logout", s.logout)
	return mux
}

func (s *AuthService) login(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&creds)

	var userID string
	err := s.db.QueryRow("SELECT id FROM users WHERE username=$1 AND password=$2",
		creds.Username, creds.Password).Scan(&userID)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	token := s.issueJWT(userID)
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (s *AuthService) refresh(w http.ResponseWriter, r *http.Request) {
	userID, err := s.validateJWT(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"token": s.issueJWT(userID)})
}

func (s *AuthService) logout(w http.ResponseWriter, r *http.Request) {
	s.db.Exec("INSERT INTO revoked_tokens(token) VALUES($1)", r.Header.Get("Authorization"))
	w.WriteHeader(http.StatusNoContent)
}

func (s *AuthService) issueJWT(userID string) string       { return "" }
func (s *AuthService) validateJWT(t string) (string, error) { return "", nil }

func main() {
	svc := NewAuthService("postgres://localhost/authdb", "secret")
	http.ListenAndServe(":8081", svc.routes())
}
