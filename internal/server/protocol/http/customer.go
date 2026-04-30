package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	intLogger "github.com/majidgolshadi/url-shortner/internal/infrastructure/logger"
)

type (
	RegisterCustomerResponse struct {
		AuthToken string `json:"auth_token"`
	}

	CustomerRegistrationService interface {
		Register(ctx context.Context) (*domain.Customer, error)
	}

	CustomerHandler struct {
		service CustomerRegistrationService
		logger  *logrus.Entry
	}
)

func NewCustomerHandler(service CustomerRegistrationService, logger *logrus.Entry) *CustomerHandler {
	return &CustomerHandler{service: service, logger: logger}
}

func (h *CustomerHandler) registerHandle(resp http.ResponseWriter, req *http.Request) {
	log := intLogger.WithContext(req.Context(), h.logger).WithField("handler", "register_customer")
	log.Info("processing customer registration request")

	customer, err := h.service.Register(req.Context())
	if err != nil {
		log.WithError(err).Error("customer registration failed")
		h.internalServerError(err, resp)
		return
	}

	log.WithField("customer_id", customer.ID).Info("customer registered successfully")
	resp.WriteHeader(http.StatusCreated)
	// nolint:errcheck
	json.NewEncoder(resp).Encode(&RegisterCustomerResponse{AuthToken: customer.AuthToken})
}

func (h *CustomerHandler) internalServerError(err error, resp http.ResponseWriter) {
	resp.WriteHeader(http.StatusInternalServerError)
	// nolint:errcheck
	json.NewEncoder(resp).Encode(&InternalServerError{Message: err.Error()})
}
