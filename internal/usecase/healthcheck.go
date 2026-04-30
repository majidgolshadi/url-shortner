package usecase

import (
	"context"
	"os"
	"time"
)

// HealthCheckService holds healthcheck datas
type HealthCheckService struct {
	hostname    string
	extraChecks map[string]HealthCheck
	// livenessFlag enables graceful drain: set it to false to stop traffic before shutdown
	// without terminating in-flight requests immediately.
	livenessFlag bool
}

func NewHealthCheckService() *HealthCheckService {
	hostname, _ := os.Hostname()

	return &HealthCheckService{
		hostname:     hostname,
		livenessFlag: true,
		extraChecks:  make(map[string]HealthCheck),
	}
}

// HealthCheck is alias for health check function
type HealthCheck func(context.Context) (bool, interface{})

func (hc *HealthCheckService) IsHealthy() (bool, map[string]interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	overallStatus := true
	overallReport := make(map[string]interface{})

	for name, healthcheck := range hc.extraChecks {
		isHealthy, report := healthcheck(ctx)
		overallReport[name] = report

		if !isHealthy {
			overallStatus = false
		}
	}

	return overallStatus, overallReport
}
