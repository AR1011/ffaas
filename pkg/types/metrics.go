package types

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

type RuntimeMetric interface {
	GetMetricType() MetricType
	GetID() uuid.UUID
}

type MetricType string

const (
	EndpointMetricType MetricType = "endpoint"
	TaskMetricType     MetricType = "task"
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

// TaskRuntimeMetric holds information about a single runtime execution.
type TaskRuntimeMetric struct {
	ID        uuid.UUID     `json:"id"`
	Type      MetricType    `json:"type"`
	TaskID    uuid.UUID     `json:"task_id"`
	DeployID  uuid.UUID     `json:"deploy_id"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
}

func (c TaskRuntimeMetric) GetMetricType() MetricType {
	return TaskMetricType
}

func (c TaskRuntimeMetric) GetID() uuid.UUID {
	return c.ID
}

func DecodeMsgpackRuntimeMetric(data []byte) (RuntimeMetric, error) {
	type unknownRuntimeMetric struct {
		Type MetricType `msgpack:"type"`
	}

	var unknown unknownRuntimeMetric
	if err := msgpack.Unmarshal(data, &unknown); err != nil {
		return nil, err
	}

	var (
		metric RuntimeMetric
		err    error
	)

	switch unknown.Type {
	case EndpointMetricType:
		m := &EndpointRuntimeMetric{}
		err = msgpack.Unmarshal(data, m)
		metric = m

	case TaskMetricType:
		m := &TaskRuntimeMetric{}
		err = msgpack.Unmarshal(data, m)
		metric = m

	default:
		err = fmt.Errorf(fmt.Sprintf("unknown metric type (%s)", unknown.Type))
	}

	return metric, err
}

// ensure implemets
var (
	_ RuntimeMetric = &EndpointRuntimeMetric{}
	_ RuntimeMetric = &TaskRuntimeMetric{}
)
