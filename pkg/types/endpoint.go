package types

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

type Endpoint struct {
	ID             uuid.UUID         `json:"id"`
	AppType        AppType           `json:"app_type"`
	Name           string            `json:"name"`
	URL            string            `json:"url"`
	Runtime        string            `json:"runtime"`
	ActiveDeployID uuid.UUID         `json:"active_deploy_id"`
	Environment    map[string]string `json:"environment"`
	DeployHistory  []*EndpointDeploy `json:"deploy_history"`
	CreatedAT      time.Time         `json:"created_at"`
}

func (e Endpoint) HasActiveDeploy() bool {
	return e.ActiveDeployID.String() != "00000000-0000-0000-0000-000000000000"
}

func (e Endpoint) GetActiveDeployID() uuid.UUID {
	return e.ActiveDeployID
}

func (e Endpoint) GetID() uuid.UUID {
	return e.ID
}

func (e Endpoint) GetAppType() AppType {
	return AppTypeEndpoint
}

func NewEndpoint(name string, runtime string, env map[string]string) *Endpoint {
	if env == nil {
		env = make(map[string]string)
	}
	id := uuid.New()
	return &Endpoint{
		ID:            id,
		AppType:       AppTypeEndpoint,
		Name:          name,
		Environment:   env,
		URL:           "",
		Runtime:       runtime,
		DeployHistory: []*EndpointDeploy{},
		CreatedAT:     time.Now(),
	}
}

type EndpointUpdateParams struct {
	Environment    map[string]string
	ActiveDeployID uuid.UUID
	Deploys        []*EndpointDeploy
}

type EndpointDeploy struct {
	ID         uuid.UUID `json:"id"`
	EndpointID uuid.UUID `json:"endpoint_id"`
	DeployType AppType   `json:"deploy_type"`
	Hash       string    `json:"hash"`
	Blob       []byte    `json:"-"`
	CreatedAT  time.Time `json:"created_at"`
}

func (e EndpointDeploy) GetID() uuid.UUID {
	return e.ID
}

func (e EndpointDeploy) GetParentID() uuid.UUID {
	return e.EndpointID
}

func (e EndpointDeploy) GetDeployType() AppType {
	return AppTypeEndpoint
}

func NewEndpointDeploy(endpoint *Endpoint, blob []byte) *EndpointDeploy {
	hashBytes := md5.Sum(blob)
	hashstr := hex.EncodeToString(hashBytes[:])
	deployID := uuid.New()
	return &EndpointDeploy{
		ID:         deployID,
		EndpointID: endpoint.ID,
		Blob:       blob,
		Hash:       hashstr,
		CreatedAT:  time.Now(),
	}
}

// ensure implements
var (
	_ App    = &Endpoint{}
	_ Deploy = &EndpointDeploy{}
)
