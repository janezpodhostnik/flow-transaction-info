package main

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/flow-go/engine/execution/computation"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/state"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"

	reusableRuntime "github.com/onflow/flow-go/fvm/runtime"
)

type RemoteDebugger struct {
	vm   *fvm.VirtualMachine
	ctx  fvm.Context
	view state.View
}

func NewRemoteDebugger(
	view *RemoteView,
	chain flow.Chain,
	directory string,
	logger zerolog.Logger) *RemoteDebugger {
	vm := fvm.NewVirtualMachine()

	// no signature processor here
	// TODO Maybe we add fee-deduction step as well
	ctx := fvm.NewContext(
		fvm.WithLogger(logger),
		fvm.WithChain(chain),
		fvm.WithTransactionProcessors(fvm.NewTransactionInvoker()),
		fvm.WithReusableCadenceRuntimePool(
			reusableRuntime.NewReusableCadenceRuntimePool(
				computation.ReusableCadenceRuntimePoolSize,
				runtime.Config{
					AccountLinkingEnabled: true,
				},
			),
		))

	return &RemoteDebugger{
		ctx:  ctx,
		vm:   vm,
		view: view,
	}
}

// RunTransaction runs the transaction given the latest sealed block data
func (d *RemoteDebugger) RunTransaction(txBody *flow.TransactionBody) (txErr, processError error) {
	blockCtx := fvm.NewContextFromParent(d.ctx, fvm.WithBlockHeader(d.ctx.BlockHeader))
	tx := fvm.Transaction(txBody, 0)
	err := d.vm.Run(blockCtx, tx, d.view)
	if err != nil {
		return nil, err
	}
	return tx.Err, nil
}

func (d *RemoteDebugger) RunScript(script *fvm.ScriptProcedure) (value cadence.Value, scriptError, processError error) {
	scriptCtx := fvm.NewContextFromParent(d.ctx, fvm.WithBlockHeader(d.ctx.BlockHeader))
	err := d.vm.Run(scriptCtx, script, d.view)
	if err != nil {
		return nil, nil, err
	}
	return script.Value, script.Err, nil
}
