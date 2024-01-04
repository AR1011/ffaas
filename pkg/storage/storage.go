package storage

import (
	"github.com/anthdm/run/pkg/types"
	"github.com/google/uuid"
)

// type Store interface {
// 	CreateEndpoint(*types.Endpoint) error
// 	UpdateEndpoint(uuid.UUID, UpdateEndpointParams) error
// 	GetEndpoint(uuid.UUID) (*types.Endpoint, error)
// 	CreateDeploy(*types.EndpointDeploy) error
// 	GetDeploy(uuid.UUID) (*types.EndpointDeploy, error)
// }

type Store interface {
	CreateApp(types.App) error
	UpdateApp(uuid.UUID, types.AppUpdateParams) error
	GetApp(uuid.UUID) (types.App, error)
	CreateDeploy(types.Deploy) error
	GetDeploy(uuid.UUID) (types.Deploy, error)
}

type MetricStore interface {
	CreateMetric(types.RuntimeMetric) error
	GetMetrics(uuid.UUID) ([]types.RuntimeMetric, error)
}
