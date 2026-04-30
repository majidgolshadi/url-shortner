package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type customerServiceMock struct {
	result *domain.Customer
	err    error
}

func (m *customerServiceMock) Register(_ context.Context) (*domain.Customer, error) {
	return m.result, m.err
}

func getCustomerHandler(svc CustomerRegistrationService) *CustomerHandler {
	return NewCustomerHandler(svc, logrus.NewEntry(logrus.StandardLogger()))
}

func TestRegisterHandle_Success(t *testing.T) {
	svc := &customerServiceMock{
		result: &domain.Customer{ID: "cid-1", AuthToken: "token-abc"},
	}
	hdl := getCustomerHandler(svc)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/customer", strings.NewReader(""))
	hdl.registerHandle(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.Contains(t, resp.Body.String(), `"auth_token":"token-abc"`)
}

func TestRegisterHandle_Error(t *testing.T) {
	svc := &customerServiceMock{err: errors.New("db failure")}
	hdl := getCustomerHandler(svc)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/customer", strings.NewReader(""))
	hdl.registerHandle(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), `"message"`)
}
