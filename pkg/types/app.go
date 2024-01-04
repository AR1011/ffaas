package types

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

type App interface {
	HasActiveDeploy() bool
	GetActiveDeployID() uuid.UUID
	GetID() uuid.UUID
	GetAppType() AppType
}

type AppUpdateParams interface{}

type AppType string

const (
	AppTypeEndpoint AppType = "endpoint"
	AppTypeTask     AppType = "task"
	AppTypeProcess  AppType = "process"
)

func IsValidAppType(s string) bool {
	switch s {
	case string(AppTypeEndpoint), string(AppTypeTask), string(AppTypeProcess):
		return true
	default:
		return false
	}
}

func ParseType(s string) AppType {
	switch s {
	case string(AppTypeEndpoint):
		return AppTypeEndpoint
	case string(AppTypeTask):
		return AppTypeTask
	case string(AppTypeProcess):
		return AppTypeProcess
	default:
		return ""
	}
}

func DecodeMsgpakApp(data []byte) (App, error) {
	type unknownApp struct {
		AppType AppType `json:"app_type"`
	}
	var uApp unknownApp

	if err := msgpack.Unmarshal(data, &uApp); err != nil {
		return nil, err
	}

	var (
		app App
		err error
	)

	switch uApp.AppType {
	case AppTypeEndpoint:
		a := &Endpoint{}
		err = msgpack.Unmarshal(data, a)
		app = a

	case AppTypeTask:
		a := &Task{}
		err = msgpack.Unmarshal(data, a)
		app = a

	case AppTypeProcess:
		a := &Process{}
		err = msgpack.Unmarshal(data, a)
		app = a

	}

	return app, err
}

func DecodeJsonApp(data []byte) (App, error) {
	type unknownApp struct {
		AppType AppType `json:"app_type"`
	}
	var uApp unknownApp

	if err := json.Unmarshal(data, &uApp); err != nil {
		return nil, err
	}

	var (
		app App
		err error
	)

	switch uApp.AppType {
	case AppTypeEndpoint:
		a := &Endpoint{}
		err = json.Unmarshal(data, a)
		app = a

	case AppTypeTask:
		a := &Task{}
		err = json.Unmarshal(data, a)
		app = a

	case AppTypeProcess:
		a := &Process{}
		err = json.Unmarshal(data, a)
		app = a

	}

	return app, err
}

var Runtimes = map[string]bool{
	"js": true,
	"go": true,
}

func ValidRuntime(runtime string) bool {
	_, ok := Runtimes[runtime]
	return ok
}
