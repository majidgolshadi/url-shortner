package url

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/id"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
)

type repositoryMock struct {
	callCount     int
	errorIndex    int
	saveErrorList map[int]error
}

func (mock *repositoryMock) Save(_ context.Context, _ *domain.URL) error {
	mock.callCount++
	err := mock.saveErrorList[mock.errorIndex]
	mock.errorIndex++
	return err
}

func (mock *repositoryMock) Delete(_ context.Context, _ string) error {
	return nil
}

func (mock *repositoryMock) Fetch(_ context.Context, _ string) (*domain.URL, error) {
	return nil, nil
}

type generatorMock struct{}

func (mock *generatorMock) GetToken(_ uint) string {
	return "token"
}

func testLogger() *logrus.Entry {
	return logrus.NewEntry(logrus.StandardLogger())
}

func newTestIDManager() *id.Manager {
	mng, _ := id.NewManager(context.Background(), id.NewInMemoryRangeManager(1), testLogger())
	return mng
}

func TestAddURLSuccessfulSave(t *testing.T) {
	repo := &repositoryMock{
		saveErrorList: map[int]error{},
	}
	idMng := newTestIDManager()
	tokenGen := &generatorMock{}

	s := NewService(idMng, tokenGen, repo, testLogger())

	_, err := s.Add(context.Background(), "sample-url", nil)
	assert.Equal(t, 1, repo.callCount)
	assert.NoError(t, err)
}

func TestAddURLSuccessfulSaveAfterTwoRetry(t *testing.T) {
	repo := &repositoryMock{
		saveErrorList: map[int]error{
			0: intErr.RepositoryDuplicateTokenErr,
			1: intErr.RepositoryDuplicateTokenErr,
			2: nil,
		},
	}
	idMng := newTestIDManager()
	tokenGen := &generatorMock{}

	s := NewService(idMng, tokenGen, repo, testLogger())
	_, err := s.Add(context.Background(), "sample-url", nil)
	assert.Equal(t, 3, repo.callCount)
	assert.NoError(t, err)
}

func TestAddURLFailedAfterMaxRetry(t *testing.T) {
	repo := &repositoryMock{
		saveErrorList: map[int]error{
			0: intErr.RepositoryDuplicateTokenErr,
			1: intErr.RepositoryDuplicateTokenErr,
			2: intErr.RepositoryDuplicateTokenErr,
		},
	}
	idMng := newTestIDManager()
	tokenGen := &generatorMock{}

	s := NewService(idMng, tokenGen, repo, testLogger())
	_, err := s.Add(context.Background(), "sample-url", nil)
	assert.Equal(t, 3, repo.callCount)
	assert.Error(t, err)
	assert.ErrorIs(t, err, intErr.RepositoryDuplicateTokenErr)
}

func TestAddURLFailedReceiveNonConflictError(t *testing.T) {
	repo := &repositoryMock{
		saveErrorList: map[int]error{
			0: errors.New("unknown error"),
		},
	}
	idMng := newTestIDManager()
	tokenGen := &generatorMock{}

	s := NewService(idMng, tokenGen, repo, testLogger())
	_, err := s.Add(context.Background(), "sample-url", nil)
	assert.Equal(t, 1, repo.callCount)
	assert.Error(t, err)
}

func TestAddURLSuccessfulSaveWithHeaders(t *testing.T) {
	repo := &repositoryMock{
		saveErrorList: map[int]error{},
	}
	idMng := newTestIDManager()
	tokenGen := &generatorMock{}

	s := NewService(idMng, tokenGen, repo, testLogger())

	headers := map[string]string{
		"X-Custom-Auth": "abc123",
		"X-Source":      "campaign-1",
	}
	_, err := s.Add(context.Background(), "sample-url", headers)
	assert.Equal(t, 1, repo.callCount)
	assert.NoError(t, err)
}
