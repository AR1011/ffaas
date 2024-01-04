package api

import (
	"encoding/json"
	"fmt"

	"github.com/anthdm/run/pkg/types"
)

type CreateParams interface {
	validate() error
	getType() types.AppType
}

// endpoint params
type CreateEndpointParams struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Runtime     string            `json:"runtime"`
	Environment map[string]string `json:"environment"`
}

func (p CreateEndpointParams) getType() types.AppType {
	return types.AppTypeEndpoint
}

func (p CreateEndpointParams) validate() error {
	minlen, maxlen := 3, 50
	if len(p.Name) < minlen {
		return fmt.Errorf("endpoint name should be at least %d characters long", minlen)
	}
	if len(p.Name) > maxlen {
		return fmt.Errorf("endpoint name can be maximum %d characters long", maxlen)
	}

	if !types.ValidRuntime(p.Runtime) {
		return fmt.Errorf("invalid runtime (%s)", p.Runtime)
	}

	return nil
}

// process params
type CreateProcessparams struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Runtime     string            `json:"runtime"`
	Environment map[string]string `json:"environment"`
}

func (p CreateProcessparams) getType() types.AppType {
	return types.AppTypeProcess
}

func (p CreateProcessparams) validate() error {
	minlen, maxlen := 3, 50
	if len(p.Name) < minlen {
		return fmt.Errorf("process name should be at least %d characters long", minlen)
	}
	if len(p.Name) > maxlen {
		return fmt.Errorf("process name can be maximum %d characters long", maxlen)
	}

	if !types.ValidRuntime(p.Runtime) {
		return fmt.Errorf("invalid runtime (%s)", p.Runtime)
	}

	return nil
}

type CreateTaskParams struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Runtime     string            `json:"runtime"`
	Interval    int64             `json:"interval"`
	Environment map[string]string `json:"environment"`
}

func (p CreateTaskParams) getType() types.AppType {
	return types.AppTypeTask
}

func (p CreateTaskParams) validate() error {
	minlen, maxlen := 3, 50
	if len(p.Name) < minlen {
		return fmt.Errorf("endpoint name should be at least %d characters long", minlen)
	}
	if len(p.Name) > maxlen {
		return fmt.Errorf("endpoint name can be maximum %d characters long", maxlen)
	}

	if !types.ValidRuntime(p.Runtime) {
		return fmt.Errorf("invalid runtime (%s)", p.Runtime)
	}

	if p.Interval == 0 {
		return fmt.Errorf("interval cannot be 0")
	}

	return nil
}

var (
	_ CreateParams = CreateEndpointParams{}
	_ CreateParams = CreateProcessparams{}
	_ CreateParams = CreateTaskParams{}
)

func DecodeParams(body []byte) (CreateParams, error) {

	type unknownParams struct {
		Type string `json:"type"`
	}

	var params unknownParams
	if err := json.Unmarshal(body, &params); err != nil {
		return nil, err
	}

	var (
		p   CreateParams
		err error
	)

	switch types.ParseType(params.Type) {
	case types.AppTypeEndpoint:
		var params CreateEndpointParams
		err = json.Unmarshal(body, &params)
		p = params

	case types.AppTypeTask:
		var params CreateTaskParams
		err = json.Unmarshal(body, &params)
		p = params

	case types.AppTypeProcess:
		var params CreateProcessparams
		err = json.Unmarshal(body, &params)
		p = params

	default:
		return nil, fmt.Errorf("invalid application type (%T)", params.Type)

	}

	if err != nil {
		return nil, err
	}

	if err := p.validate(); err != nil {
		return nil, err
	}

	return p, nil

}
