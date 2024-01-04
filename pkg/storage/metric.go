package storage

import (
	"fmt"
	"sync"

	"github.com/anthdm/run/pkg/types"
	"github.com/google/uuid"
)

type MemoryMetricStore struct {
	mu      sync.RWMutex
	metrics map[uuid.UUID][]types.RuntimeMetric
}

func NewMemoryMetricStore() *MemoryMetricStore {
	return &MemoryMetricStore{
		metrics: make(map[uuid.UUID][]types.RuntimeMetric),
	}
}

func (s *MemoryMetricStore) CreateMetric(metric types.RuntimeMetric) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var (
		metrics []types.RuntimeMetric
		ok      bool
	)
	metrics, ok = s.metrics[metric.GetID()]
	if !ok {
		metrics = make([]types.RuntimeMetric, 0)
	}

	metrics = append(metrics, metric)

	s.metrics[metric.GetID()] = metrics
	return nil
}

func (s *MemoryMetricStore) GetMetrics(id uuid.UUID) ([]types.RuntimeMetric, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	metrics, ok := s.metrics[id]
	if !ok {
		return nil, fmt.Errorf("could not find metrics for cron (%s)", id)
	}
	return metrics, nil
}

var _ MetricStore = &MemoryMetricStore{}
