package types

import "github.com/google/uuid"

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
	AppTypeCron     AppType = "cron"
	AppTypeProcess  AppType = "process"
)

func IsValidAppType(s string) bool {
	switch s {
	case string(AppTypeEndpoint), string(AppTypeCron), string(AppTypeProcess):
		return true
	default:
		return false
	}
}

func ParseType(s string) AppType {
	switch s {
	case string(AppTypeEndpoint):
		return AppTypeEndpoint
	case string(AppTypeCron):
		return AppTypeCron
	case string(AppTypeProcess):
		return AppTypeProcess
	default:
		return ""
	}
}

var Runtimes = map[string]bool{
	"js": true,
	"go": true,
}

func ValidRuntime(runtime string) bool {
	_, ok := Runtimes[runtime]
	return ok
}
