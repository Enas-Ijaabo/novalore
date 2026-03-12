package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Gateway struct {
	authServiceURL    string
	paymentServiceURL string
}

func (g *Gateway) proxyTo(target string) http.HandlerFunc {
	dst, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(dst)
	return proxy.ServeHTTP
}

func (g *Gateway) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/", g.proxyTo(g.authServiceURL))
	mux.HandleFunc("/payments/", g.proxyTo(g.paymentServiceURL))
	return mux
}

func main() {
	gw := &Gateway{
		authServiceURL:    "http://auth-service:8081",
		paymentServiceURL: "http://payment-service:8082",
	}
	http.ListenAndServe(":8080", gw.routes())
}
