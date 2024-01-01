package types

import (
	"time"

	"github.com/google/uuid"
)

type RuntimeMetric interface {
	GetMetricType() MetricType
	GetID() uuid.UUID
}

type MetricType string

const (
	EndpointMetricType MetricType = "endpoint"
	CronMetricType     MetricType = "cron"
)

// EndpointRuntimeMetric holds information about a single runtime execution.
type EndpointRuntimeMetric struct {
	ID         uuid.UUID     `json:"id"`
	Type       MetricType    `json:"type"`
	EndpointID uuid.UUID     `json:"endpoint_id"`
	DeployID   uuid.UUID     `json:"deploy_id"`
	RequestURL string        `json:"request_url"`
	Duration   time.Duration `json:"duration"`
	StartTime  time.Time     `json:"start_time"`
}

func (e EndpointRuntimeMetric) GetMetricType() MetricType {
	return EndpointMetricType
}

func (e EndpointRuntimeMetric) GetID() uuid.UUID {
	return e.ID
}

// CronRuntimeMetric holds information about a single runtime execution.
type CronRuntimeMetric struct {
	ID        uuid.UUID     `json:"id"`
	Type      MetricType    `json:"type"`
	CronID    uuid.UUID     `json:"cron_id"`
	DeployID  uuid.UUID     `json:"deploy_id"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
}

func (c CronRuntimeMetric) GetMetricType() MetricType {
	return CronMetricType
}

func (c CronRuntimeMetric) GetID() uuid.UUID {
	return c.ID
}

// ensure implemets
var (
	_ RuntimeMetric = &EndpointRuntimeMetric{}
	_ RuntimeMetric = &CronRuntimeMetric{}
)
