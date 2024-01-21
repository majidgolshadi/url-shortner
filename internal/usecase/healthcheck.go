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
	// livenessFlag used to store/change the apprlication readiness and liveness state
	// false - the application is not ready yet
	// true - The application is ready to serve requests
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
	ctx, cancle := context.WithTimeout(context.Background(), time.Second)
	defer cancle()

	overalStatus := true
	overalReport := make(map[string]interface{})

	for name, healthcheck := range hc.extraChecks {
		isHealthy, report := healthcheck(ctx)
		overalReport[name] = report

		if !isHealthy {
			overalStatus = false
		}
	}

	return overalStatus, overalReport
}
