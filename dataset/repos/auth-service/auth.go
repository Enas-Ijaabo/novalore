package main

import (
	"database/sql"
	"errors"
	"net/http"
	"time"
)

const (
	jwtExpiry          = 24 * time.Hour
	maxLoginAttempts   = 5
	lockoutDuration    = 15 * time.Minute
	tokenRefreshWindow = 30 * time.Minute
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrAccountLocked      = errors.New("account locked after too many failed attempts")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenRevoked       = errors.New("token has been revoked")
)

type AuthService struct {
	db *sql.DB
}

func (a *AuthService) Login(username, password string) (string, error) {
	var attempts int
	var locked bool
	a.db.QueryRow(
		"SELECT login_attempts, locked FROM users WHERE username = $1",
		username,
	).Scan(&attempts, &locked)

	if locked {
		return "", ErrAccountLocked
	}
	if attempts >= maxLoginAttempts {
		a.db.Exec("UPDATE users SET locked = true WHERE username = $1", username)
		return "", ErrAccountLocked
	}

	var hash string
	err := a.db.QueryRow(
		"SELECT password_hash FROM users WHERE username = $1", username,
	).Scan(&hash)
	if err != nil || !checkHash(password, hash) {
		a.db.Exec("UPDATE users SET login_attempts = login_attempts + 1 WHERE username = $1", username)
		return "", ErrInvalidCredentials
	}

	a.db.Exec("UPDATE users SET login_attempts = 0 WHERE username = $1", username)
	token := generateJWT(username, jwtExpiry)
	return token, nil
}

func (a *AuthService) ValidateToken(token string) (string, error) {
	var revoked bool
	a.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM revoked_tokens WHERE token = $1)", token,
	).Scan(&revoked)
	if revoked {
		return "", ErrTokenRevoked
	}

	username, expiry, err := parseJWT(token)
	if err != nil || time.Now().After(expiry) {
		return "", ErrTokenExpired
	}
	return username, nil
}

func (a *AuthService) Logout(token string) error {
	_, err := a.db.Exec(
		"INSERT INTO revoked_tokens (token, revoked_at) VALUES ($1, $2)",
		token, time.Now(),
	)
	return err
}

func (a *AuthService) RefreshToken(token string) (string, error) {
	username, expiry, err := parseJWT(token)
	if err != nil {
		return "", ErrTokenExpired
	}
	if time.Until(expiry) > tokenRefreshWindow {
		return "", errors.New("token is not eligible for refresh yet")
	}
	newToken := generateJWT(username, jwtExpiry)
	a.db.Exec("INSERT INTO revoked_tokens (token, revoked_at) VALUES ($1, $2)", token, time.Now())
	return newToken, nil
}

func setupRoutes(svc *AuthService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", svc.loginHandler)
	mux.HandleFunc("/auth/refresh", svc.refreshHandler)
	mux.HandleFunc("/auth/logout", svc.logoutHandler)
	mux.HandleFunc("/auth/validate", svc.validateHandler)
	return mux
}

func checkHash(password, hash string) bool        { return false }
func generateJWT(username string, d time.Duration) string { return "" }
func parseJWT(token string) (string, time.Time, error)    { return "", time.Time{}, nil }
func (a *AuthService) loginHandler(w http.ResponseWriter, r *http.Request)    {}
func (a *AuthService) refreshHandler(w http.ResponseWriter, r *http.Request)  {}
func (a *AuthService) logoutHandler(w http.ResponseWriter, r *http.Request)   {}
func (a *AuthService) validateHandler(w http.ResponseWriter, r *http.Request) {}
