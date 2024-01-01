package types

import (
	"github.com/google/uuid"
)

type Deploy interface {
	GetID() uuid.UUID
	GetParentID() uuid.UUID
	GetDeployType() AppType
}
