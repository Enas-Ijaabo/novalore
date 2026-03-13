package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	requestTimeout = 30 * time.Second
	maxBodySize    = 1 << 20 // 1MB
	rateLimitRPM   = 100     // requests per minute per IP
)

var routes = map[string]string{
	"/auth/":     "http://auth-service:8081",
	"/payments/": "http://payment-service:8082",
}

type Gateway struct {
	proxies map[string]*httputil.ReverseProxy
}

func NewGateway() *Gateway {
	g := &Gateway{proxies: make(map[string]*httputil.ReverseProxy)}
	for prefix, target := range routes {
		u, _ := url.Parse(target)
		g.proxies[prefix] = httputil.NewSingleHostReverseProxy(u)
	}
	return g
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	for prefix, proxy := range g.proxies {
		if strings.HasPrefix(r.URL.Path, prefix) {
			proxy.ServeHTTP(w, r)
			return
		}
	}
	http.Error(w, "not found", http.StatusNotFound)
}
