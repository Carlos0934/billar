package app

import (
	"context"

	"github.com/Carlos0934/billar/internal/core"
)

type HealthDTO struct {
	Name   string `json:"name" toon:"name"`
	Status string `json:"status" toon:"status"`
}

type HealthService struct {
	appName string
}

func NewHealthService(appName string) HealthService {
	return HealthService{appName: appName}
}

func (s HealthService) Status(ctx context.Context) (HealthDTO, error) {
	_ = ctx

	health := core.NewHealth(s.appName)

	return HealthDTO{
		Name:   health.Name,
		Status: health.Status,
	}, nil
}
