package url

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	"github.com/majidgolshadi/url-shortner/internal/id"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
)

type datastoreMock struct {
	callCount int
	errorIndex int
	saveErrorList map[int]error
}

func (mock *datastoreMock) Save(ctx context.Context, url *domain.Url) error {
	mock.callCount++
	err := mock.saveErrorList[mock.errorIndex]
	mock.errorIndex++

	return err
}

func (mock *datastoreMock) Delete(ctx context.Context, token string) error {
	return nil
}

func (mock *datastoreMock) Fetch(ctx context.Context, token string) (*domain.Url, error) {
	return nil, nil
}

type generatorMock struct {

}

func (mock *generatorMock) GetToken(id uint) string {
	return "token"
}

func getIdManager() *id.Manager {
	mng, _ := id.NewManager(context.Background(), id.NewInMemoryRangeManager(1))
	return mng
}



func TestAddUrlSuccessfulSave(t *testing.T) {
	db := &datastoreMock{
		saveErrorList: map[int]error{},
	}
	idMng := getIdManager()
	tokenGen := &generatorMock{}

	s := NewService(idMng, tokenGen, db)

	err := s.AddUrl(context.Background(), "sample-url")
	assert.Equal(t, 1, db.callCount)
	assert.Nil(t, err)
}

func TestAddUrlSuccessfulSaveAfterTwoRetry(t *testing.T) {
	db := &datastoreMock{
		saveErrorList: map[int]error{
			0: intErr.RepositoryDuplicateTokenErr,
			1: intErr.RepositoryDuplicateTokenErr,
			2: nil,
		},
	}
	idMng := getIdManager()
	tokenGen := &generatorMock{}

	s := NewService(idMng, tokenGen, db)
	err := s.AddUrl(context.Background(), "sample-url")
	assert.Equal(t, 3, db.callCount)
	assert.Nil(t, err)
}

func TestAddUrlFailedAfterMaxRetry(t *testing.T) {
	db := &datastoreMock{
		saveErrorList: map[int]error{
			0: intErr.RepositoryDuplicateTokenErr,
			1: intErr.RepositoryDuplicateTokenErr,
			2: intErr.RepositoryDuplicateTokenErr,
		},
	}
	idMng := getIdManager()
	tokenGen := &generatorMock{}

	s := NewService(idMng, tokenGen, db)
	err := s.AddUrl(context.Background(), "sample-url")
	assert.Equal(t, 3, db.callCount)
	assert.Equal(t, err, intErr.RepositoryDuplicateTokenErr)
}

func TestAddUrlFailedReciveNoneConflictError(t *testing.T) {
	db := &datastoreMock{
		saveErrorList: map[int]error{
			0: errors.New("unknown error"),
		},
	}
	idMng := getIdManager()
	tokenGen := &generatorMock{}

	s := NewService(idMng, tokenGen, db)
	err := s.AddUrl(context.Background(), "sample-url")
	assert.Equal(t, 1, db.callCount)
	assert.NotNil(t, err)
}