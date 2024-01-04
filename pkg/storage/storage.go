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
	GetApps() ([]types.App, error)
	CreateDeploy(types.Deploy) error
	GetDeploy(uuid.UUID) (types.Deploy, error)
}

type MetricStore interface {
	CreateRuntimeMetric(types.RuntimeMetric) error
	GetRuntimeMetrics(uuid.UUID) ([]types.RuntimeMetric, error)
}
