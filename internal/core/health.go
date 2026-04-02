package core

import "strings"

const HealthyStatus = "ok"

type Health struct {
	Name   string
	Status string
}

func NewHealth(name string) Health {
	if strings.TrimSpace(name) == "" {
		name = "billar"
	}

	return Health{
		Name:   name,
		Status: HealthyStatus,
	}
}
