package http

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/majidgolshadi/url-shortner/internal/usecase"
)

type HealthCheckResponse struct {
	HTTPStatus int                    `json:"http-status"`
	Version    HealthCheckVersion     `json:"version"`
	Time       int64                  `json:"time"`
	Status     bool                   `json:"status"`
	Host       string                 `json:"hostname"`
	Services   map[string]interface{} `json:"checks"`
}

type HealthCheckVersion struct {
	Tag    string `json:"tag"`
	Commit string `json:"commit"`
}

type healthCheckHandler struct {
	tag                string
	commitHash         string
	logger             *logrus.Entry
	healthCheckService *usecase.HealthCheckService
}

func NewHealthCheckHandler(tag string, commitHash string, logger *logrus.Entry, healthCheckService *usecase.HealthCheckService) *healthCheckHandler {
	return &healthCheckHandler{
		tag:                tag,
		commitHash:         commitHash,
		logger:             logger,
		healthCheckService: healthCheckService,
	}
}

func (hc *healthCheckHandler) Handle(resp http.ResponseWriter, req *http.Request) {
	overallStatus, mapServices := hc.healthCheckService.IsHealthy()
	hostname, _ := os.Hostname()

	statusCode := http.StatusOK
	if !overallStatus {
		statusCode = http.StatusInternalServerError
	}

	responseBody := &HealthCheckResponse{
		HTTPStatus: statusCode,
		Version: HealthCheckVersion{
			Tag:    hc.tag,
			Commit: hc.commitHash,
		},
		Time:     time.Now().UnixMilli(),
		Status:   overallStatus,
		Host:     hostname,
		Services: mapServices,
	}

	resp.WriteHeader(statusCode)
	if encodeErr := json.NewEncoder(resp).Encode(responseBody); encodeErr != nil {
		hc.logger.Errorf("healthcheck encoding response error: %v", encodeErr)
	}
}
