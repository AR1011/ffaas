package storage

import (
	"fmt"
	"sync"

	"github.com/anthdm/ffaas/pkg/types"
	"github.com/google/uuid"
)

type MemoryStore struct {
	mu      sync.RWMutex
	apps    map[uuid.UUID]types.App
	deploys map[uuid.UUID]types.Deploy
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		apps:    make(map[uuid.UUID]types.App),
		deploys: make(map[uuid.UUID]types.Deploy),
	}
}

func (s *MemoryStore) CreateApp(app types.App) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ensureIsApp(app); err != nil {
		return err
	}
	s.apps[app.GetID()] = app
	return nil
}

func (s *MemoryStore) GetApp(id uuid.UUID) (types.App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	app, ok := s.apps[id]
	if !ok {
		return nil, fmt.Errorf("could not find app with id (%s)", id)
	}
	return app, nil

}

func (s *MemoryStore) UpdateApp(id uuid.UUID, params types.AppUpdateParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, err := s.GetApp(id)
	if err != nil {
		return err
	}

	switch p := params.(type) {
	case *types.EndpointUpdateParams:
		endpoint, ok := app.(*types.Endpoint)
		if !ok {
			return fmt.Errorf("invalid params for app type %T. Want: %T  Got: %T", app, &types.EndpointUpdateParams{}, p)
		}
		return s.updateEndpoint(endpoint, p)

	case *types.CronUpdateParams:
		cron, ok := app.(*types.Cron)
		if !ok {
			return fmt.Errorf("invalid params for app type %T. Want: %T  Got: %T", app, &types.CronUpdateParams{}, p)
		}
		return s.updateCron(cron, p)

	case *types.ProcessUpdateParams:
		process, ok := app.(*types.Process)
		if !ok {
			return fmt.Errorf("invalid params for app type %T. Want: %T  Got: %T", app, &types.ProcessUpdateParams{}, p)
		}
		return s.updateProcess(process, p)

	default:
		return fmt.Errorf("unknown params type (%T)", params)
	}

}

func (s *MemoryStore) updateEndpoint(e *types.Endpoint, params *types.EndpointUpdateParams) error {

	if params.ActiveDeployID.String() != "00000000-0000-0000-0000-000000000000" {
		e.ActiveDeployID = params.ActiveDeployID
	}

	if params.Environment != nil {
		for key, val := range params.Environment {
			e.Environment[key] = val
		}
	}

	if len(params.Deploys) > 0 {
		e.DeployHistory = append(e.DeployHistory, params.Deploys...)
	}

	return nil
}

func (s *MemoryStore) updateProcess(p *types.Process, params *types.ProcessUpdateParams) error {

	if params.ActiveDeployID.String() != "00000000-0000-0000-0000-000000000000" {
		p.ActiveDeployID = params.ActiveDeployID
	}

	if params.Environment != nil {
		for key, val := range params.Environment {
			p.Environment[key] = val
		}
	}

	if len(params.Deploys) > 0 {
		p.DeployHistory = append(p.DeployHistory, params.Deploys...)
	}

	return nil
}

func (s *MemoryStore) updateCron(c *types.Cron, params *types.CronUpdateParams) error {

	if params.ActiveDeployID.String() != "00000000-0000-0000-0000-000000000000" {
		c.ActiveDeployID = params.ActiveDeployID
	}

	if params.Environment != nil {
		for key, val := range params.Environment {
			c.Environment[key] = val
		}
	}

	if len(params.Deploys) > 0 {
		c.DeployHistory = append(c.DeployHistory, params.Deploys...)
	}

	if params.Interval != 0 {
		c.Interval = params.Interval
	}

	return nil
}

func (s *MemoryStore) CreateDeploy(deploy types.Deploy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deploys[deploy.GetID()] = deploy

	return nil
}

func (s *MemoryStore) GetDeploy(id uuid.UUID) (types.Deploy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deploy, ok := s.deploys[id]
	if !ok {
		return nil, fmt.Errorf("could not find deploy with id (%s)", id)
	}

	return deploy, nil

}

func ensureIsApp(app types.App) error {
	switch app.(type) {
	case *types.Endpoint:
		return nil
	case *types.Cron:
		return nil
	case *types.Process:
		return nil
	default:
		return fmt.Errorf("unknown app type")
	}
}

// ensure implements
var _ Store = &MemoryStore{}
