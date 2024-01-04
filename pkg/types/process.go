package types

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

type Process struct {
	ID             uuid.UUID         `json:"id"`
	AppType        AppType           `json:"app_type"`
	Name           string            `json:"name"`
	Runtime        string            `json:"runtime"`
	ActiveDeployID uuid.UUID         `json:"active_deploy_id"`
	Environment    map[string]string `json:"environment"`
	DeployHistory  []*ProcessDeploy  `json:"deploy_history"`
	CreatedAT      time.Time         `json:"created_at"`
}

func (p Process) HasActiveDeploy() bool {
	return p.ActiveDeployID.String() != "00000000-0000-0000-0000-000000000000"
}

func (p Process) GetActiveDeployID() uuid.UUID {
	return p.ActiveDeployID
}

func (p Process) GetID() uuid.UUID {
	return p.ID
}

func (p Process) GetAppType() AppType {
	return AppTypeProcess
}

func NewProcess(name string, runtime string, env map[string]string) *Process {
	if env == nil {
		env = make(map[string]string)
	}
	id := uuid.New()
	return &Process{
		ID:            id,
		AppType:       AppTypeProcess,
		Name:          name,
		Runtime:       runtime,
		Environment:   env,
		DeployHistory: []*ProcessDeploy{},
		CreatedAT:     time.Now(),
	}
}

type ProcessUpdateParams struct {
	Environment    map[string]string
	ActiveDeployID uuid.UUID
	Deploys        []*ProcessDeploy
}

type ProcessDeploy struct {
	ID        uuid.UUID `json:"id"`
	ProcessID uuid.UUID `json:"process_id"`
	Hash      string    `json:"hash"`
	Blob      []byte    `json:"-"`
	CreatedAT time.Time `json:"created_at"`
}

func (p ProcessDeploy) GetID() uuid.UUID {
	return p.ID
}

func (p ProcessDeploy) GetParentID() uuid.UUID {
	return p.ProcessID
}

func (p ProcessDeploy) GetDeployType() AppType {
	return AppTypeProcess
}

func NewProcessDeploy(process *Process, blob []byte) *ProcessDeploy {
	hashBytes := md5.Sum(blob)
	hashstr := hex.EncodeToString(hashBytes[:])
	deployID := uuid.New()
	return &ProcessDeploy{
		ID:        deployID,
		ProcessID: process.ID,
		Blob:      blob,
		Hash:      hashstr,
		CreatedAT: time.Now(),
	}
}

// ensure implements
var (
	_ App    = &Process{}
	_ Deploy = &ProcessDeploy{}
)
