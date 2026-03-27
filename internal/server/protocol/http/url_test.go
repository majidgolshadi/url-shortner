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
)

type urlServiceMock struct{}

func (mock *urlServiceMock) Add(_ context.Context, url string, _ map[string]string) (string, error) {
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
		}, nil
	}
	return nil, errors.New("dummy error")
}

func getURLHandler() *URLHandler {
	return NewURLHandler(&urlServiceMock{}, logrus.NewEntry(logrus.StandardLogger()))
}

func TestAddUrlHandle(t *testing.T) {
	urlHdl := getURLHandler()

	tests := map[string]struct {
		requestBody            string
		expectedRespStatusCode int
		expectedRespBody       string
	}{
		"none json request": {
			requestBody:            "",
			expectedRespStatusCode: http.StatusBadRequest,
		},
		"empty url request": {
			requestBody:            `{"unknown_key":"value"}`,
			expectedRespStatusCode: http.StatusBadRequest,
		},
		"valid request - successful": {
			requestBody:            `{"url":"http://successful-url.com"}`,
			expectedRespStatusCode: http.StatusOK,
			expectedRespBody:       "{\"token\":\"token\"}\n",
		},
		"valid request with headers - successful": {
			requestBody:            `{"url":"http://successful-url.com","headers":{"X-Custom":"value"}}`,
			expectedRespStatusCode: http.StatusOK,
			expectedRespBody:       "{\"token\":\"token\"}\n",
		},
		"valid request - internal server error": {
			requestBody:            `{"url":"http://fail-url.com"}`,
			expectedRespStatusCode: http.StatusInternalServerError,
			expectedRespBody:       "{\"message\":\"dummy error\"}\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.requestBody))
			urlHdl.addUrlHandle(resp, req)
			assert.Equal(t, test.expectedRespStatusCode, resp.Code)
			assert.Equal(t, test.expectedRespBody, resp.Body.String())
		})
	}
}

func TestFetchUrlHandle(t *testing.T) {
	urlHdl := getURLHandler()

	tests := map[string]struct {
		token                  string
		expectedRespStatusCode int
		expectedRespBody       string
	}{
		"url without token": {
			token:                  "",
			expectedRespStatusCode: http.StatusBadRequest,
		},
		"valid request - successful": {
			token:                  "successful-token",
			expectedRespStatusCode: http.StatusOK,
			expectedRespBody:       "{\"url\":\"http://successful-url.com\",\"token\":\"token\",\"headers\":{\"X-Custom-Header\":\"custom-value\"}}\n",
		},
		"valid request - internal server error": {
			token:                  "fail-token",
			expectedRespStatusCode: http.StatusInternalServerError,
			expectedRespBody:       "{\"message\":\"dummy error\"}\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
			req = mux.SetURLVars(req, map[string]string{
				"token": test.token,
			})
			urlHdl.fetchUrlHandle(resp, req)
			assert.Equal(t, test.expectedRespStatusCode, resp.Code)
			assert.Equal(t, test.expectedRespBody, resp.Body.String())
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

func TestDeleteUrlHandle(t *testing.T) {
	urlHdl := getURLHandler()

	tests := map[string]struct {
		token                  string
		expectedRespStatusCode int
		expectedRespBody       string
	}{
		"url without token": {
			token:                  "",
			expectedRespStatusCode: http.StatusBadRequest,
		},
		"valid request - successful": {
			token:                  "successful-token",
			expectedRespStatusCode: http.StatusAccepted,
		},
		"valid request - internal server error": {
			token:                  "fail-token",
			expectedRespStatusCode: http.StatusInternalServerError,
			expectedRespBody:       "{\"message\":\"dummy error\"}\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/", strings.NewReader(""))
			req = mux.SetURLVars(req, map[string]string{
				"token": test.token,
			})
			urlHdl.deleteUrlHandle(resp, req)
			assert.Equal(t, test.expectedRespStatusCode, resp.Code)
			assert.Equal(t, test.expectedRespBody, resp.Body.String())
		})
	}
}