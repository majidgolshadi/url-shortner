package customer

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	intLogger "github.com/majidgolshadi/url-shortner/internal/infrastructure/logger"
	"github.com/majidgolshadi/url-shortner/internal/storage"
)

// Service handles customer business logic.
type Service struct {
	repo   storage.CustomerRepository
	logger *logrus.Entry
}

// NewService creates a new customer service.
func NewService(repo storage.CustomerRepository, logger *logrus.Entry) *Service {
	return &Service{repo: repo, logger: logger}
}

// Register creates a new customer with a cryptographically random auth token.
func (s *Service) Register(ctx context.Context) (*domain.Customer, error) {
	log := intLogger.WithContext(ctx, s.logger)
	log.Info("registering new customer")

	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		log.WithError(err).Error("failed to generate auth token")
		return nil, err
	}

	customer := &domain.Customer{
		ID:        uuid.New().String(),
		AuthToken: base64.RawURLEncoding.EncodeToString(buf[:]),
	}

	if err := s.repo.Save(ctx, customer); err != nil {
		log.WithError(err).Error("failed to save customer")
		return nil, err
	}

	log.WithField("customer_id", customer.ID).Info("customer registered successfully")
	return customer, nil
}

// FindByAuthToken looks up a customer by their auth token.
func (s *Service) FindByAuthToken(ctx context.Context, authToken string) (*domain.Customer, error) {
	return s.repo.FindByAuthToken(ctx, authToken)
}
