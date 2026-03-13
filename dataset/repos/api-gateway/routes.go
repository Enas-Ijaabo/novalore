package main

// Route table — all external traffic enters here.
// Services must not be called directly from outside the cluster.
// Adding a new service requires a route entry here and nowhere else.

var serviceRoutes = []Route{
	{Prefix: "/auth/", Target: "http://auth-service:8081", RequiresAuth: false},
	{Prefix: "/payments/", Target: "http://payment-service:8082", RequiresAuth: true},
}

type Route struct {
	Prefix       string
	Target       string
	RequiresAuth bool
}
