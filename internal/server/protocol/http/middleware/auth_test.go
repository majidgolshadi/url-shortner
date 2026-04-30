package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type customerFinderMock struct {
	result *domain.Customer
	err    error
}

func (m *customerFinderMock) FindByAuthToken(_ context.Context, _ string) (*domain.Customer, error) {
	return m.result, m.err
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestAuth_MissingHeader(t *testing.T) {
	middleware := Auth(&customerFinderMock{})
	handler := middleware(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestAuth_WrongFormat(t *testing.T) {
	middleware := Auth(&customerFinderMock{})
	handler := middleware(http.HandlerFunc(okHandler))

	tests := []string{
		"token-without-bearer-prefix",
		"Basic dXNlcjpwYXNz",
		"Bearer",
		"Bearer ",
	}

	for _, header := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", header)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusUnauthorized, resp.Code, "header: %s", header)
	}
}

func TestAuth_FindError(t *testing.T) {
	mock := &customerFinderMock{err: http.ErrNoCookie}
	mw := Auth(mock)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-looking-token")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestAuth_ValidToken(t *testing.T) {
	customer := &domain.Customer{ID: "cid-1", AuthToken: "valid-token"}
	mock := &customerFinderMock{result: customer}
	mw := Auth(mock)

	var capturedCustomer *domain.Customer
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCustomer, _ = CustomerFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := mw(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, customer, capturedCustomer)
}
