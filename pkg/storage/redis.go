package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/anthdm/run/pkg/types"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(opts *redis.Options) (*RedisStore, error) {
	if opts == nil {
		opts = &redis.Options{}
	}

	client := redis.NewClient(opts)
	err := client.Ping(context.Background()).Err()
	if err != nil {
		err = fmt.Errorf("failed to connect to the Redis server: %s", err)
		return nil, err
	}

	return &RedisStore{
		client: client,
	}, nil
}

func (s *RedisStore) CreateApp(app types.App) error {
	b, err := msgpack.Marshal(app)
	if err != nil {
		return err
	}
	return s.client.Set(context.Background(), app.GetID().String(), b, 0).Err()
}

func (s *RedisStore) UpdateApp(id uuid.UUID, params types.AppUpdateParams) error {
	app, err := s.GetApp(id)
	if err != nil {
		return err
	}

	switch p := params.(type) {
	case types.EndpointUpdateParams:
		endpoint, ok := app.(*types.Endpoint)
		if !ok {
			return fmt.Errorf("invalid params for app type %T. Want: %T  Got: %T", app, &types.EndpointUpdateParams{}, p)
		}
		return s.updateEndpoint(endpoint, p)

	case types.TaskUpdateParams:
		task, ok := app.(*types.Task)
		if !ok {
			return fmt.Errorf("invalid params for app type %T. Want: %T  Got: %T", app, &types.TaskUpdateParams{}, p)
		}
		return s.updateTask(task, p)

	case types.ProcessUpdateParams:
		process, ok := app.(*types.Process)
		if !ok {
			return fmt.Errorf("invalid params for app type %T. Want: %T  Got: %T", app, &types.ProcessUpdateParams{}, p)
		}
		return s.updateProcess(process, p)

	default:
		return fmt.Errorf("unknown params type (%T)", params)
	}

}

func (s *RedisStore) updateEndpoint(e *types.Endpoint, params types.EndpointUpdateParams) error {
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

	b, err := msgpack.Marshal(e)
	if err != nil {
		return err
	}
	return s.client.Set(context.Background(), e.ID.String(), b, 0).Err()
}

func (s *RedisStore) updateProcess(p *types.Process, params types.ProcessUpdateParams) error {
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

	b, err := msgpack.Marshal(p)
	if err != nil {
		return err
	}
	return s.client.Set(context.Background(), p.ID.String(), b, 0).Err()
}

func (s *RedisStore) updateTask(c *types.Task, params types.TaskUpdateParams) error {
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

	b, err := msgpack.Marshal(c)
	if err != nil {
		return err
	}

	return s.client.Set(context.Background(), c.ID.String(), b, 0).Err()
}

func (s *RedisStore) GetApp(id uuid.UUID) (types.App, error) {

	b, err := s.client.Get(context.Background(), id.String()).Bytes()
	if err != nil {
		return nil, err
	}

	type unknownApp struct {
		AppType types.AppType
	}

	var app unknownApp
	err = msgpack.Unmarshal(b, &app)
	if err != nil {
		return nil, err
	}

	switch app.AppType {
	case types.AppTypeEndpoint:
		endpoint := &types.Endpoint{}
		err = msgpack.Unmarshal(b, endpoint)
		return endpoint, err
	case types.AppTypeTask:
		task := &types.Task{}
		err = msgpack.Unmarshal(b, task)
		return task, err
	case types.AppTypeProcess:
		process := &types.Process{}
		err = msgpack.Unmarshal(b, process)
		return process, err

	default:
		return nil, fmt.Errorf("unknown app type (%s)", app.AppType)
	}
}

func (s *RedisStore) GetApps() ([]types.App, error) {
	keys, err := s.client.Keys(context.Background(), "*").Result()
	if err != nil {
		return nil, err
	}

	apps := make([]types.App, len(keys))
	for i, key := range keys {
		app, err := s.GetApp(uuid.MustParse(key))
		if err != nil {
			return nil, err
		}
		apps[i] = app
	}

	return apps, nil
}

func (s *RedisStore) CreateDeploy(deploy types.Deploy) error {
	b, err := msgpack.Marshal(deploy)
	if err != nil {
		return err
	}
	return s.client.Set(context.Background(), deploy.GetID().String(), b, 0).Err()
}

func (s *RedisStore) GetDeploy(id uuid.UUID) (types.Deploy, error) {
	b, err := s.client.Get(context.Background(), id.String()).Bytes()
	if err != nil {
		return nil, err
	}

	type unknownDeploy struct {
		DeployType types.AppType
	}

	var deploy unknownDeploy
	err = msgpack.Unmarshal(b, &deploy)
	if err != nil {
		return nil, err
	}

	switch deploy.DeployType {
	case types.AppTypeEndpoint:
		endpointDeploy := &types.EndpointDeploy{}
		err = msgpack.Unmarshal(b, endpointDeploy)
		return endpointDeploy, err

	case types.AppTypeTask:
		taskDeploy := &types.TaskDeploy{}
		err = msgpack.Unmarshal(b, taskDeploy)
		return taskDeploy, err

	case types.AppTypeProcess:
		processDeploy := &types.ProcessDeploy{}
		err = msgpack.Unmarshal(b, processDeploy)
		return processDeploy, err

	default:
		return nil, fmt.Errorf("unknown deploy type (%s)", deploy.DeployType)
	}
}

func (s *RedisStore) CreateRuntimeMetric(metric types.RuntimeMetric) error {
	b, err := msgpack.Marshal(metric)
	if err != nil {
		return err
	}
	return s.client.Set(context.Background(), metric.GetID().String(), b, 0).Err()
}

func (s *RedisStore) GetRuntimeMetrics() ([]types.RuntimeMetric, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys, err := s.client.Keys(ctx, "*").Result()
	if err != nil {
		return nil, err
	}

	pipeline := s.client.Pipeline()
	for _, key := range keys {
		pipeline.Get(ctx, key)
	}

	cmds, err := pipeline.Exec(ctx)
	if err != nil {
		return nil, err
	}

	metrics := make([]types.RuntimeMetric, len(cmds))

	for i, cmd := range cmds {
		b, err := cmd.(*redis.StringCmd).Bytes()
		if err != nil {
			continue
		}
		m, err := types.DecodeMsgpackRuntimeMetric(b)
		if err != nil {
			continue
		}
		metrics[i] = m
	}

	return metrics, nil

}

// ensure implements
var _ Store = &RedisStore{}
