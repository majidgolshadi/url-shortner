package customer

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type customerRepoMock struct {
	saveErr         error
	findResult      *domain.Customer
	findErr         error
}

func (m *customerRepoMock) Save(_ context.Context, _ *domain.Customer) error {
	return m.saveErr
}

func (m *customerRepoMock) FindByAuthToken(_ context.Context, _ string) (*domain.Customer, error) {
	return m.findResult, m.findErr
}

func testLogger() *logrus.Entry {
	return logrus.NewEntry(logrus.StandardLogger())
}

func TestRegister_Success(t *testing.T) {
	repo := &customerRepoMock{}
	svc := NewService(repo, testLogger())

	customer, err := svc.Register(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, customer)
	assert.NotEmpty(t, customer.ID)
	assert.NotEmpty(t, customer.AuthToken)
	assert.Len(t, customer.AuthToken, 43)
}

func TestRegister_SaveError(t *testing.T) {
	repo := &customerRepoMock{saveErr: errors.New("db error")}
	svc := NewService(repo, testLogger())

	customer, err := svc.Register(context.Background())

	assert.Error(t, err)
	assert.Nil(t, customer)
}

func TestFindByAuthToken_Success(t *testing.T) {
	expected := &domain.Customer{ID: "cid-1", AuthToken: "tok-1"}
	repo := &customerRepoMock{findResult: expected}
	svc := NewService(repo, testLogger())

	result, err := svc.FindByAuthToken(context.Background(), "tok-1")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestFindByAuthToken_NotFound(t *testing.T) {
	repo := &customerRepoMock{findErr: errors.New("not found")}
	svc := NewService(repo, testLogger())

	result, err := svc.FindByAuthToken(context.Background(), "missing")

	assert.Error(t, err)
	assert.Nil(t, result)
}
