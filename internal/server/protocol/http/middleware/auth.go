package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/majidgolshadi/url-shortner/internal/domain"
)

// ContextKey is the type used for context keys in this package.
type ContextKey string

// TestCustomerContextKey is exported so tests in other packages can inject a customer into context.
const TestCustomerContextKey ContextKey = "customer"

const customerContextKey = TestCustomerContextKey

// CustomerFinder looks up a customer by their auth token.
type CustomerFinder interface {
	FindByAuthToken(ctx context.Context, authToken string) (*domain.Customer, error)
}

// Auth returns middleware that validates Bearer tokens and injects the customer into the request context.
func Auth(finder CustomerFinder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				w.WriteHeader(http.StatusUnauthorized)
				// nolint:errcheck
				json.NewEncoder(w).Encode(map[string]string{"message": "unauthorized"})
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				w.WriteHeader(http.StatusUnauthorized)
				// nolint:errcheck
				json.NewEncoder(w).Encode(map[string]string{"message": "unauthorized"})
				return
			}

			customer, err := finder.FindByAuthToken(r.Context(), token)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				// nolint:errcheck
				json.NewEncoder(w).Encode(map[string]string{"message": "unauthorized"})
				return
			}

			ctx := context.WithValue(r.Context(), customerContextKey, customer)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CustomerFromContext retrieves the authenticated customer from the request context.
func CustomerFromContext(ctx context.Context) (*domain.Customer, bool) {
	c, ok := ctx.Value(customerContextKey).(*domain.Customer)
	return c, ok
}
