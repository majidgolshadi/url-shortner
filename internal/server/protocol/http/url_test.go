package http

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type urlServiceMock struct {
}

func (mock *urlServiceMock) Add(ctx context.Context, url string) (token string, insertError error) {
	if url == "http://successful-url.com" {
		return "token", nil
	}

	return "", errors.New("dummy error")
}

func (mock *urlServiceMock) Delete(ctx context.Context, token string) error {
	if token == "successful-token" {
		return nil
	}

	return errors.New("dummy error")
}

func (mock *urlServiceMock) Fetch(ctx context.Context, token string) (*domain.Url, error) {
	if token == "successful-token" {
		return &domain.Url{
			UrlPath: "http://successful-url.com",
			Token:   "token",
		}, nil
	}

	return nil, errors.New("dummy error")
}

func getUrlHandler() *UrlHandler {
	return NewUrlHandler(&urlServiceMock{})
}

func TestAddUrlHandle(t *testing.T) {
	urlHdl := getUrlHandler()

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
			requestBody:            "{\"unknown_key\":\"value\"}",
			expectedRespStatusCode: http.StatusBadRequest,
		},
		"valid request - successful": {
			requestBody:            "{\"url\":\"http://successful-url.com\"}",
			expectedRespStatusCode: http.StatusOK,
			expectedRespBody:       "{\"token\":\"token\"}\n",
		},
		"valid request - internal server error": {
			requestBody:            "{\"url\":\"http://fail-url.com\"}",
			expectedRespStatusCode: http.StatusInternalServerError,
			expectedRespBody:       "{\"message\":\"dummy error\"}\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(test.requestBody))
			urlHdl.addUrlHandle(resp, req)
			assert.Equal(t, test.expectedRespStatusCode, resp.Code)
			assert.Equal(t, test.expectedRespBody, resp.Body.String())
		})
	}
}

func TestFetchUrlHandle(t *testing.T) {
	urlHdl := getUrlHandler()

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
			expectedRespBody:       "{\"url\":\"http://successful-url.com\",\"token\":\"http://successful-url.com\"}\n",
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
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
			req = mux.SetURLVars(req, map[string]string{
				"token": test.token,
			})
			urlHdl.fetchUrlHandle(resp, req)
			assert.Equal(t, test.expectedRespStatusCode, resp.Code)
			assert.Equal(t, test.expectedRespBody, resp.Body.String())
		})
	}
}

func TestDeleteUrlHandle(t *testing.T) {
	urlHdl := getUrlHandler()

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
