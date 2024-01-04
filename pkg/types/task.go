package types

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID             uuid.UUID         `json:"id"`
	AppType        AppType           `json:"app_type"`
	Name           string            `json:"name"`
	Runtime        string            `json:"runtime"`
	ActiveDeployID uuid.UUID         `json:"active_deploy_id"`
	Environment    map[string]string `json:"environment"`
	DeployHistory  []*TaskDeploy     `json:"deploy_history"`
	CreatedAT      time.Time         `json:"created_at"`
	Interval       int64             `json:"interval"` // interval in seconds
}

func (c Task) HasActiveDeploy() bool {
	return c.ActiveDeployID.String() != "00000000-0000-0000-0000-000000000000"
}

func (c Task) GetActiveDeployID() uuid.UUID {
	return c.ActiveDeployID
}

func (c Task) GetAppType() AppType {
	return AppTypeTask
}

func (c Task) GetID() uuid.UUID {
	return c.ID
}

func NewTask(name string, runtime string, interval int64, env map[string]string) *Task {
	if env == nil {
		env = make(map[string]string)
	}
	id := uuid.New()
	return &Task{
		ID:            id,
		AppType:       AppTypeTask,
		Name:          name,
		Runtime:       runtime,
		Environment:   env,
		DeployHistory: []*TaskDeploy{},
		CreatedAT:     time.Now(),
		Interval:      interval,
	}
}

type TaskUpdateParams struct {
	Environment    map[string]string
	ActiveDeployID uuid.UUID
	Deploys        []*TaskDeploy
	Interval       int64 // interval in seconds
}

type TaskDeploy struct {
	ID         uuid.UUID `json:"id"`
	TaskID     uuid.UUID `json:"task_id"`
	DeployType AppType   `json:"deploy_type"`
	Hash       string    `json:"hash"`
	Blob       []byte    `json:"-"`
	CreatedAT  time.Time `json:"created_at"`
}

func NewTaskDeploy(task *Task, blob []byte) *TaskDeploy {
	hashBytes := md5.Sum(blob)
	hashstr := hex.EncodeToString(hashBytes[:])
	deployID := uuid.New()
	return &TaskDeploy{
		ID:         deployID,
		TaskID:     task.ID,
		DeployType: AppTypeTask,
		Blob:       blob,
		Hash:       hashstr,
		CreatedAT:  time.Now(),
	}
}

func (c TaskDeploy) GetID() uuid.UUID {
	return c.ID
}

func (c TaskDeploy) GetParentID() uuid.UUID {
	return c.TaskID
}

func (c TaskDeploy) GetDeployType() AppType {
	return AppTypeTask
}

// ensure implements
var (
	_ App    = &Task{}
	_ Deploy = &TaskDeploy{}
)
