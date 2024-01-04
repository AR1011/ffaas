package types

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

type Cron struct {
	ID             uuid.UUID         `json:"id"`
	AppType        AppType           `json:"app_type"`
	Name           string            `json:"name"`
	Runtime        string            `json:"runtime"`
	ActiveDeployID uuid.UUID         `json:"active_deploy_id"`
	Environment    map[string]string `json:"environment"`
	DeployHistory  []*CronDeploy     `json:"deploy_history"`
	CreatedAT      time.Time         `json:"created_at"`
	Interval       int64             `json:"interval"` // interval in seconds
}

func (c Cron) HasActiveDeploy() bool {
	return c.ActiveDeployID.String() != "00000000-0000-0000-0000-000000000000"
}

func (c Cron) GetActiveDeployID() uuid.UUID {
	return c.ActiveDeployID
}

func (c Cron) GetAppType() AppType {
	return AppTypeCron
}

func (c Cron) GetID() uuid.UUID {
	return c.ID
}

func NewCron(name string, runtime string, interval int64, env map[string]string) *Cron {
	if env == nil {
		env = make(map[string]string)
	}
	id := uuid.New()
	return &Cron{
		ID:            id,
		AppType:       AppTypeCron,
		Name:          name,
		Runtime:       runtime,
		Environment:   env,
		DeployHistory: []*CronDeploy{},
		CreatedAT:     time.Now(),
		Interval:      interval,
	}
}

type CronUpdateParams struct {
	Environment    map[string]string
	ActiveDeployID uuid.UUID
	Deploys        []*CronDeploy
	Interval       int64 // interval in seconds
}

type CronDeploy struct {
	ID         uuid.UUID `json:"id"`
	CronID     uuid.UUID `json:"cron_id"`
	DeployType AppType   `json:"deploy_type"`
	Hash       string    `json:"hash"`
	Blob       []byte    `json:"-"`
	CreatedAT  time.Time `json:"created_at"`
}

func NewCronDeploy(cron *Cron, blob []byte) *CronDeploy {
	hashBytes := md5.Sum(blob)
	hashstr := hex.EncodeToString(hashBytes[:])
	deployID := uuid.New()
	return &CronDeploy{
		ID:         deployID,
		CronID:     cron.ID,
		DeployType: AppTypeCron,
		Blob:       blob,
		Hash:       hashstr,
		CreatedAT:  time.Now(),
	}
}

func (c CronDeploy) GetID() uuid.UUID {
	return c.ID
}

func (c CronDeploy) GetParentID() uuid.UUID {
	return c.CronID
}

func (c CronDeploy) GetDeployType() AppType {
	return AppTypeCron
}

// ensure implements
var (
	_ App    = &Cron{}
	_ Deploy = &CronDeploy{}
)
