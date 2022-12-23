package main

import (
	"fmt"
	"github.com/janezpodhostnik/flow-transaction-info/registers"
	"github.com/onflow/flow-go/fvm/state"
	"github.com/onflow/flow-go/model/flow"
)

type RemoteView struct {
	Parent *RemoteView
	Delta  map[string]flow.RegisterValue

	getRemoteRegister registers.RegisterGetRegisterFunc
}

func NewRemoteView(getRemoteRegister registers.RegisterGetRegisterFunc) *RemoteView {

	view := &RemoteView{
		Delta:             make(map[string]flow.RegisterValue),
		getRemoteRegister: getRemoteRegister,
	}
	return view
}

func (v *RemoteView) NewChild() state.View {
	return &RemoteView{
		Parent: v,
		Delta:  make(map[string][]byte),
	}
}

func (v *RemoteView) MergeView(o state.View) error {
	var other *RemoteView
	var ok bool
	if other, ok = o.(*RemoteView); !ok {
		return fmt.Errorf("can not merge: view type mismatch (given: %T, expected:RemoteView)", o)
	}

	for k, value := range other.Delta {
		v.Delta[k] = value
	}
	return nil
}

func (v *RemoteView) DropDelta() {
	v.Delta = make(map[string]flow.RegisterValue)
}

func (v *RemoteView) Set(owner, key string, value flow.RegisterValue) error {
	v.Delta[owner+"~"+key] = value
	return nil
}

func (v *RemoteView) Get(owner, key string) (flow.RegisterValue, error) {

	// first check the delta
	value, found := v.Delta[owner+"~"+key]
	if found {
		return value, nil
	}

	// then call the parent (if exist)
	if v.Parent != nil {
		return v.Parent.Get(owner, key)
	}

	// last use the getRemoteRegister
	resp, err := v.getRemoteRegister(owner, key)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// returns all the registers that has been touched
func (v *RemoteView) AllRegisters() []flow.RegisterID {
	panic("Not implemented yet")
}

func (v *RemoteView) RegisterUpdates() ([]flow.RegisterID, []flow.RegisterValue) {
	panic("Not implemented yet")
}

func (v *RemoteView) Touch(owner, key string) error {
	// no-op for now
	return nil
}

func (v *RemoteView) Delete(owner, key string) error {
	v.Delta[owner+"~"+key] = nil
	return nil
}
