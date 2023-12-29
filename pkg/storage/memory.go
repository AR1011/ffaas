package storage

import (
	"fmt"
	"strings"
	"sync"

	"github.com/anthdm/ffaas/pkg/types"
	"github.com/anthdm/ffaas/pkg/utils"
	"github.com/google/uuid"
)

type MemoryStore struct {
	mu      sync.RWMutex
	apps    map[uuid.UUID]*types.Application
	deploys map[uuid.UUID]*types.Deploy
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		apps:    make(map[uuid.UUID]*types.Application),
		deploys: make(map[uuid.UUID]*types.Deploy),
	}
}

func (s *MemoryStore) CreateApplication(app *types.Application) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apps[app.ID] = app
	return nil
}

func (s *MemoryStore) GetApplication(id uuid.UUID) (*types.Application, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.apps[id]
	if !ok {
		return nil, fmt.Errorf("could not find app with id (%s)", id)
	}
	return app, nil
}

type UpdateAppParams struct {
	Environment    map[string]string
	ActiveDeployID uuid.UUID
}

func (s *MemoryStore) UpdateApplication(id uuid.UUID, params UpdateAppParams) error {
	app, err := s.GetApplication(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if params.ActiveDeployID.String() != "00000000-0000-0000-0000-000000000000" {
		app.ActiveDeployID = params.ActiveDeployID
	}
	if params.Environment != nil {
		for key, val := range params.Environment {
			app.Environment[key] = val
		}
	}
	return nil
}

func (s *MemoryStore) CreateDeploy(deploy *types.Deploy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deploys[deploy.ID] = deploy
	return nil
}

func (s *MemoryStore) GetDeploy(id uuid.UUID) (*types.Deploy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	deploy, ok := s.deploys[id]
	if !ok {
		return nil, fmt.Errorf("could not find deployment with id (%s)", id)
	}
	return deploy, nil
}

func (s *MemoryStore) AppendApplicationLogs(appID uuid.UUID, stdout string, stderr string) error {
	stdoutFile, stderrFile, err := utils.GetStdioByAppID(appID)
	if err != nil {
		return err
	}

	stdout += "\n"
	stderr += "\n"

	_, err = stdoutFile.Write([]byte(stdout))
	if err != nil {
		return err
	}

	_, err = stderrFile.Write([]byte(stderr))
	if err != nil {
		return err
	}

	return nil
}

func (s *MemoryStore) GetApplicationLogs(appID uuid.UUID) (*types.Logs, error) {
	logs := &types.Logs{}

	stdout, stderr, err := utils.ReadStdioByAppID(appID)
	if err != nil {
		return logs, err
	}

	// temp just for better viewing
	logs.Stderr = strings.Split(stderr, "\n")
	logs.Stdout = strings.Split(stdout, "\n")

	// remove empty strings
	utils.RemoveEmptyStrings(&logs.Stderr)
	utils.RemoveEmptyStrings(&logs.Stdout)

	return logs, nil
}
