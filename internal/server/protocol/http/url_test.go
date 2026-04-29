package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/server/protocol/http/middleware"
)

const testCustomerID = "owner-customer-id"

type urlServiceMock struct{}

func (mock *urlServiceMock) Add(_ context.Context, url string, _ map[string]string, _ string) (string, error) {
	if url == "http://successful-url.com" {
		return "token", nil
	}
	return "", errors.New("dummy error")
}

func (mock *urlServiceMock) Delete(_ context.Context, token string) error {
	if token == "successful-token" {
		return nil
	}
	return errors.New("dummy error")
}

func (mock *urlServiceMock) Fetch(_ context.Context, token string) (*domain.URL, error) {
	if token == "successful-token" {
		return &domain.URL{
			Path:  "http://successful-url.com",
			Token: "token",
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
			},
			OgHTML:     `<meta property="og:title" content="Test Title" />`,
			CustomerID: testCustomerID,
		}, nil
	}
	if token == "no-og-token" {
		return &domain.URL{
			Path:       "http://no-og-url.com",
			Token:      "no-og-token",
			CustomerID: testCustomerID,
		}, nil
	}
	if token == "other-owner-token" {
		return &domain.URL{
			Path:       "http://other-url.com",
			Token:      "other-owner-token",
			CustomerID: "some-other-customer-id",
		}, nil
	}
	return nil, errors.New("dummy error")
}

func (mock *urlServiceMock) RefreshOG(_ context.Context, token string) error {
	if token == "successful-token" {
		return nil
	}
	return errors.New("dummy error")
}

func getURLHandler() *URLHandler {
	return NewURLHandler(&urlServiceMock{}, logrus.NewEntry(logrus.StandardLogger()))
}

// withCustomer injects a customer into the request context (simulates auth middleware).
func withCustomer(req *http.Request, customerID string) *http.Request {
	customer := &domain.Customer{ID: customerID}
	ctx := context.WithValue(req.Context(), middleware.TestCustomerContextKey, customer)
	return req.WithContext(ctx)
}

func TestAddUrlHandle(t *testing.T) {
	urlHdl := getURLHandler()

	tests := map[string]struct {
		requestBody            string
		injectCustomer         bool
		expectedRespStatusCode int
		expectedRespBody       string
	}{
		"no customer in context (no auth)": {
			requestBody:            `{"url":"http://successful-url.com"}`,
			injectCustomer:         false,
			expectedRespStatusCode: http.StatusUnauthorized,
		},
		"none json request": {
			requestBody:            "",
			injectCustomer:         true,
			expectedRespStatusCode: http.StatusBadRequest,
		},
		"empty url request": {
			requestBody:            `{"unknown_key":"value"}`,
			injectCustomer:         true,
			expectedRespStatusCode: http.StatusBadRequest,
		},
		"valid request - successful": {
			requestBody:            `{"url":"http://successful-url.com"}`,
			injectCustomer:         true,
			expectedRespStatusCode: http.StatusOK,
			expectedRespBody:       "{\"token\":\"token\"}\n",
		},
		"valid request with headers - successful": {
			requestBody:            `{"url":"http://successful-url.com","headers":{"X-Custom":"value"}}`,
			injectCustomer:         true,
			expectedRespStatusCode: http.StatusOK,
			expectedRespBody:       "{\"token\":\"token\"}\n",
		},
		"valid request - internal server error": {
			requestBody:            `{"url":"http://fail-url.com"}`,
			injectCustomer:         true,
			expectedRespStatusCode: http.StatusInternalServerError,
			expectedRespBody:       "{\"message\":\"dummy error\"}\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.requestBody))
			if test.injectCustomer {
				req = withCustomer(req, testCustomerID)
			}
			urlHdl.addUrlHandle(resp, req)
			assert.Equal(t, test.expectedRespStatusCode, resp.Code)
			if test.expectedRespBody != "" {
				assert.Equal(t, test.expectedRespBody, resp.Body.String())
			}
		})
	}
}

func TestFetchUrlHandle(t *testing.T) {
	urlHdl := getURLHandler()

	tests := map[string]struct {
		token                  string
		customerID             string
		injectCustomer         bool
		expectedRespStatusCode int
		expectedRespBody       string
	}{
		"no customer in context (no auth)": {
			token:                  "successful-token",
			injectCustomer:         false,
			expectedRespStatusCode: http.StatusUnauthorized,
		},
		"url without token": {
			token:                  "",
			injectCustomer:         true,
			customerID:             testCustomerID,
			expectedRespStatusCode: http.StatusBadRequest,
		},
		"valid request - successful": {
			token:                  "successful-token",
			injectCustomer:         true,
			customerID:             testCustomerID,
			expectedRespStatusCode: http.StatusOK,
			expectedRespBody:       "{\"url\":\"http://successful-url.com\",\"token\":\"token\",\"headers\":{\"X-Custom-Header\":\"custom-value\"}}\n",
		},
		"valid request - internal server error": {
			token:                  "fail-token",
			injectCustomer:         true,
			customerID:             testCustomerID,
			expectedRespStatusCode: http.StatusInternalServerError,
			expectedRespBody:       "{\"message\":\"dummy error\"}\n",
		},
		"valid request - forbidden (not owner)": {
			token:                  "other-owner-token",
			injectCustomer:         true,
			customerID:             testCustomerID,
			expectedRespStatusCode: http.StatusForbidden,
			expectedRespBody:       "{\"message\":\"forbidden\"}\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
			req = mux.SetURLVars(req, map[string]string{
				"token": test.token,
			})
			if test.injectCustomer {
				req = withCustomer(req, test.customerID)
			}
			urlHdl.fetchUrlHandle(resp, req)
			assert.Equal(t, test.expectedRespStatusCode, resp.Code)
			if test.expectedRespBody != "" {
				assert.Equal(t, test.expectedRespBody, resp.Body.String())
			}
		})
	}
}

func TestRedirectHandle(t *testing.T) {
	urlHdl := getURLHandler()

	tests := map[string]struct {
		token                  string
		expectedRespStatusCode int
		expectedHeaders        map[string]string
	}{
		"redirect without token": {
			token:                  "",
			expectedRespStatusCode: http.StatusBadRequest,
		},
		"redirect successful with custom headers": {
			token:                  "successful-token",
			expectedRespStatusCode: http.StatusFound,
			expectedHeaders: map[string]string{
				"X-Custom-Header": "custom-value",
				"Location":        "http://successful-url.com",
			},
		},
		"redirect - internal server error": {
			token:                  "fail-token",
			expectedRespStatusCode: http.StatusInternalServerError,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
			req = mux.SetURLVars(req, map[string]string{
				"token": test.token,
			})
			urlHdl.redirectHandle(resp, req)
			assert.Equal(t, test.expectedRespStatusCode, resp.Code)
			for key, value := range test.expectedHeaders {
				assert.Equal(t, value, resp.Header().Get(key))
			}
		})
	}
}

func TestRedirectHandle_BotWithOgData(t *testing.T) {
	urlHdl := getURLHandler()

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
	req.Header.Set("User-Agent", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)")
	req = mux.SetURLVars(req, map[string]string{
		"token": "successful-token",
	})

	urlHdl.redirectHandle(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header().Get("Content-Type"))
	assert.Contains(t, resp.Body.String(), `<meta property="og:title" content="Test Title" />`)
	assert.Contains(t, resp.Body.String(), `<meta http-equiv="refresh"`)
	assert.Contains(t, resp.Body.String(), `http://successful-url.com`)
}

func TestRedirectHandle_BotWithoutOgData(t *testing.T) {
	urlHdl := getURLHandler()

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
	req.Header.Set("User-Agent", "Twitterbot/1.0")
	req = mux.SetURLVars(req, map[string]string{
		"token": "no-og-token",
	})

	urlHdl.redirectHandle(resp, req)

	// Should fall back to regular redirect when no OG data
	assert.Equal(t, http.StatusFound, resp.Code)
	assert.Equal(t, "http://no-og-url.com", resp.Header().Get("Location"))
}

func TestRedirectHandle_RegularUserWithOgData(t *testing.T) {
	urlHdl := getURLHandler()

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0")
	req = mux.SetURLVars(req, map[string]string{
		"token": "successful-token",
	})

	urlHdl.redirectHandle(resp, req)

	// Regular users always get a redirect, even if OG data exists
	assert.Equal(t, http.StatusFound, resp.Code)
	assert.Equal(t, "http://successful-url.com", resp.Header().Get("Location"))
}

func TestRefreshOgHandle(t *testing.T) {
	urlHdl := getURLHandler()

	tests := map[string]struct {
		token                  string
		expectedRespStatusCode int
		expectedRespBody       string
	}{
		"missing token": {
			token:                  "",
			expectedRespStatusCode: http.StatusBadRequest,
		},
		"successful refresh": {
			token:                  "successful-token",
			expectedRespStatusCode: http.StatusAccepted,
		},
		"refresh error": {
			token:                  "fail-token",
			expectedRespStatusCode: http.StatusInternalServerError,
			expectedRespBody:       "{\"message\":\"dummy error\"}\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(""))
			req = mux.SetURLVars(req, map[string]string{
				"token": test.token,
			})
			urlHdl.refreshOgHandle(resp, req)
			assert.Equal(t, test.expectedRespStatusCode, resp.Code)
			if test.expectedRespBody != "" {
				assert.Equal(t, test.expectedRespBody, resp.Body.String())
			}
		})
	}
}

func TestDeleteUrlHandle(t *testing.T) {
	urlHdl := getURLHandler()

	tests := map[string]struct {
		token                  string
		customerID             string
		injectCustomer         bool
		expectedRespStatusCode int
		expectedRespBody       string
	}{
		"no customer in context (no auth)": {
			token:                  "successful-token",
			injectCustomer:         false,
			expectedRespStatusCode: http.StatusUnauthorized,
		},
		"url without token": {
			token:                  "",
			injectCustomer:         true,
			customerID:             testCustomerID,
			expectedRespStatusCode: http.StatusBadRequest,
		},
		"valid request - successful": {
			token:                  "successful-token",
			injectCustomer:         true,
			customerID:             testCustomerID,
			expectedRespStatusCode: http.StatusAccepted,
		},
		"valid request - fetch error": {
			token:                  "fail-token",
			injectCustomer:         true,
			customerID:             testCustomerID,
			expectedRespStatusCode: http.StatusInternalServerError,
			expectedRespBody:       "{\"message\":\"dummy error\"}\n",
		},
		"valid request - forbidden (not owner)": {
			token:                  "other-owner-token",
			injectCustomer:         true,
			customerID:             testCustomerID,
			expectedRespStatusCode: http.StatusForbidden,
			expectedRespBody:       "{\"message\":\"forbidden\"}\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/", strings.NewReader(""))
			req = mux.SetURLVars(req, map[string]string{
				"token": test.token,
			})
			if test.injectCustomer {
				req = withCustomer(req, test.customerID)
			}
			urlHdl.deleteUrlHandle(resp, req)
			assert.Equal(t, test.expectedRespStatusCode, resp.Code)
			if test.expectedRespBody != "" {
				assert.Equal(t, test.expectedRespBody, resp.Body.String())
			}
		})
	}
}
