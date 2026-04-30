package url

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/id"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
)

// repositoryMock controls each method independently via dedicated fields.
type repositoryMock struct {
	// Save fields
	callCount     int
	errorIndex    int
	saveErrorList map[int]error
	urlCount      int

	// Fetch fields
	fetchResult *domain.URL
	fetchErr    error

	// Delete fields
	deleteErr error

	// UpdateOgHTML fields
	updateOGErr  error
	updateOGDone chan struct{} // closed when UpdateOgHTML is called (for goroutine sync)
}

func (mock *repositoryMock) Save(_ context.Context, _ *domain.URL) error {
	mock.callCount++
	err := mock.saveErrorList[mock.errorIndex]
	mock.errorIndex++
	return err
}

func (mock *repositoryMock) Delete(_ context.Context, _ string) error {
	return mock.deleteErr
}

func (mock *repositoryMock) Fetch(_ context.Context, _ string) (*domain.URL, error) {
	return mock.fetchResult, mock.fetchErr
}

func (mock *repositoryMock) UpdateOgHTML(_ context.Context, _ string, _ string) error {
	if mock.updateOGDone != nil {
		defer close(mock.updateOGDone)
	}
	return mock.updateOGErr
}

func (mock *repositoryMock) CountByCustomer(_ context.Context, _ string) (int, error) {
	return mock.urlCount, nil
}

// countErrRepositoryMock returns an error from CountByCustomer.
type countErrRepositoryMock struct {
	repositoryMock
	countErr error
}

func (m *countErrRepositoryMock) CountByCustomer(_ context.Context, _ string) (int, error) {
	return 0, m.countErr
}

// errorIDProvider always returns an error from GetNextID.
type errorIDProvider struct{}

func (m *errorIDProvider) GetNextID(_ context.Context) (uint, error) {
	return 0, errors.New("id generation failed")
}

// generatorMock always returns the same token.
type generatorMock struct{}

func (mock *generatorMock) GetToken(_ uint) string {
	return "token"
}

// ogFetcherMock returns a non-empty OG HTML string.
type ogFetcherMock struct{}

func (mock *ogFetcherMock) FetchOgHTML(_ context.Context, _ string) string {
	return `<meta property="og:title" content="Test" />`
}

// emptyOgFetcherMock returns empty string and signals done via channel.
type emptyOgFetcherMock struct {
	done chan struct{}
}

func (m *emptyOgFetcherMock) FetchOgHTML(_ context.Context, _ string) string {
	defer close(m.done)
	return ""
}

// signalingOgFetcherMock returns non-empty HTML and signals done via channel.
type signalingOgFetcherMock struct {
	done chan struct{}
}

func (m *signalingOgFetcherMock) FetchOgHTML(_ context.Context, _ string) string {
	defer close(m.done)
	return `<meta property="og:title" content="Test" />`
}

func testLogger() *logrus.Entry {
	return logrus.NewEntry(logrus.StandardLogger())
}

func newTestIDManager() *id.Manager {
	mng, _ := id.NewManager(context.Background(), id.NewInMemoryRangeManager(1), testLogger())
	return mng
}

const testBudget = 100

// --- Add tests ---

func TestAddURLSuccessfulSave(t *testing.T) {
	repo := &repositoryMock{saveErrorList: map[int]error{}}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	_, err := s.Add(context.Background(), "sample-url", nil, "customer-1")
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
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	_, err := s.Add(context.Background(), "sample-url", nil, "customer-1")
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
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	_, err := s.Add(context.Background(), "sample-url", nil, "customer-1")
	assert.Equal(t, 3, repo.callCount)
	assert.Error(t, err)
	assert.ErrorIs(t, err, intErr.RepositoryDuplicateTokenErr)
}

func TestAddURLFailedReceiveNonConflictError(t *testing.T) {
	repo := &repositoryMock{
		saveErrorList: map[int]error{0: errors.New("unknown error")},
	}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	_, err := s.Add(context.Background(), "sample-url", nil, "customer-1")
	assert.Equal(t, 1, repo.callCount)
	assert.Error(t, err)
}

func TestAddURLSuccessfulSaveWithHeaders(t *testing.T) {
	repo := &repositoryMock{saveErrorList: map[int]error{}}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	headers := map[string]string{
		"X-Custom-Auth": "abc123",
		"X-Source":      "campaign-1",
	}
	_, err := s.Add(context.Background(), "sample-url", headers, "customer-1")
	assert.Equal(t, 1, repo.callCount)
	assert.NoError(t, err)
}

func TestAddURL_BudgetExceeded(t *testing.T) {
	repo := &repositoryMock{
		saveErrorList: map[int]error{},
		urlCount:      testBudget,
	}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	_, err := s.Add(context.Background(), "sample-url", nil, "customer-1")
	assert.Equal(t, 0, repo.callCount)
	assert.ErrorIs(t, err, intErr.BudgetExceededErr)
}

func TestAddURL_BudgetNotExceeded(t *testing.T) {
	repo := &repositoryMock{
		saveErrorList: map[int]error{},
		urlCount:      testBudget - 1,
	}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	_, err := s.Add(context.Background(), "sample-url", nil, "customer-1")
	assert.Equal(t, 1, repo.callCount)
	assert.NoError(t, err)
}

func TestAddURL_IDProviderError(t *testing.T) {
	repo := &repositoryMock{saveErrorList: map[int]error{}}
	s := NewService(&errorIDProvider{}, &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	_, err := s.Add(context.Background(), "sample-url", nil, "customer-1")
	assert.Equal(t, 0, repo.callCount)
	assert.Error(t, err)
}

func TestAddURL_CountByCustomerError(t *testing.T) {
	repo := &countErrRepositoryMock{
		repositoryMock: repositoryMock{saveErrorList: map[int]error{}},
		countErr:       errors.New("db error"),
	}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	_, err := s.Add(context.Background(), "sample-url", nil, "customer-1")
	assert.Equal(t, 0, repo.callCount)
	assert.Error(t, err)
}

// --- Fetch tests ---

func TestFetch_Success(t *testing.T) {
	expected := &domain.URL{Path: "http://example.com", Token: "tok", CustomerID: "cid"}
	repo := &repositoryMock{fetchResult: expected}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	result, err := s.Fetch(context.Background(), "tok")
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestFetch_Error(t *testing.T) {
	repo := &repositoryMock{fetchErr: errors.New("not found")}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	result, err := s.Fetch(context.Background(), "tok")
	assert.Error(t, err)
	assert.Nil(t, result)
}

// --- Delete tests ---

func TestDelete_Success(t *testing.T) {
	repo := &repositoryMock{}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	err := s.Delete(context.Background(), "tok")
	assert.NoError(t, err)
}

func TestDelete_Error(t *testing.T) {
	repo := &repositoryMock{deleteErr: errors.New("delete failed")}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	err := s.Delete(context.Background(), "tok")
	assert.Error(t, err)
}

// --- RefreshOG tests ---

func TestRefreshOG_Success(t *testing.T) {
	repo := &repositoryMock{
		fetchResult: &domain.URL{Path: "http://example.com", Token: "tok"},
	}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	err := s.RefreshOG(context.Background(), "tok")
	assert.NoError(t, err)
}

func TestRefreshOG_FetchError(t *testing.T) {
	repo := &repositoryMock{fetchErr: errors.New("not found")}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	err := s.RefreshOG(context.Background(), "tok")
	assert.Error(t, err)
}

func TestRefreshOG_UpdateOGError(t *testing.T) {
	repo := &repositoryMock{
		fetchResult:  &domain.URL{Path: "http://example.com", Token: "tok"},
		updateOGErr:  errors.New("update failed"),
	}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, &ogFetcherMock{}, testBudget, testLogger())

	err := s.RefreshOG(context.Background(), "tok")
	assert.Error(t, err)
}

// --- fetchAndStoreOgAsync goroutine tests ---

func TestFetchAndStoreOgAsync_EmptyOGHTML(t *testing.T) {
	done := make(chan struct{})
	fetcher := &emptyOgFetcherMock{done: done}
	repo := &repositoryMock{saveErrorList: map[int]error{}}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, fetcher, testBudget, testLogger())

	_, err := s.Add(context.Background(), "sample-url", nil, "customer-1")
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine did not complete in time")
	}
}

func TestFetchAndStoreOgAsync_UpdateOGError(t *testing.T) {
	done := make(chan struct{})
	fetcher := &signalingOgFetcherMock{done: done}
	repo := &repositoryMock{
		saveErrorList: map[int]error{},
		updateOGErr:   errors.New("update failed"),
		updateOGDone:  make(chan struct{}),
	}
	s := NewService(newTestIDManager(), &generatorMock{}, repo, fetcher, testBudget, testLogger())

	_, err := s.Add(context.Background(), "sample-url", nil, "customer-1")
	assert.NoError(t, err)

	// wait for ogFetcher to be called first
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ogFetcher goroutine did not complete in time")
	}
	// then wait for UpdateOgHTML to be called
	select {
	case <-repo.updateOGDone:
	case <-time.After(2 * time.Second):
		t.Fatal("UpdateOgHTML goroutine did not complete in time")
	}
}
