package main

import (
	"net/http"
	"os"
)

func ApiKeyAuthMiddleware(next http.Handler) http.Handler {
	apiKey := os.Getenv("API_KEY")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		token := r.Header.Get("Authorization")
		if token != apiKey {
			http.Error(w, "Invalid API Key", http.StatusForbidden)
		} else {
			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, r)
		}
	})
}
