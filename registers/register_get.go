package registers

import (
	"github.com/onflow/flow-go/model/flow"
)

type RegisterGetRegisterFunc func(string, string) (flow.RegisterValue, error)

type RegisterGetWrapper interface {
	Wrap(RegisterGetRegisterFunc) RegisterGetRegisterFunc
}
