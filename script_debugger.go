package main

//
// import (
// 	"context"
// 	"github.com/hashicorp/go-multierror"
// 	"github.com/janezpodhostnik/flow-transaction-info/registers"
// 	"github.com/onflow/cadence"
// 	"github.com/onflow/flow-archive/api/archive"
// 	"github.com/onflow/flow-go/engine/execution/state"
// 	"github.com/onflow/flow-go/fvm"
// 	"github.com/onflow/flow-go/ledger/common/pathfinder"
// 	"github.com/onflow/flow-go/ledger/complete"
// 	"github.com/onflow/flow-go/model/flow"
// 	"github.com/rs/zerolog"
// 	"io"
// 	"os"
// 	"path/filepath"
// )
//
// type ScriptDebugger struct {
// 	script      *fvm.ScriptProcedure
// 	blockHeight uint64
// 	archiveHost string
// 	chain       flow.Chain
//
// 	directory string
//
// 	log zerolog.Logger
//
// 	registerGetWrappers []registers.RegisterGetWrapper
//
// 	debugger    *RemoteDebugger
// 	logHandlers []LogHandler
// }
//
// func NewScriptDebugger(
// 	script *fvm.ScriptProcedure,
// 	blockHeight uint64,
// 	archiveHost string,
// 	chain flow.Chain,
// 	logger zerolog.Logger) *ScriptDebugger {
//
// 	return &ScriptDebugger{
// 		script:      script,
// 		blockHeight: blockHeight,
// 		archiveHost: archiveHost,
// 		chain:       chain,
//
// 		directory: "s_" + script.ID.String(),
//
// 		log: logger,
// 	}
// }
//
// func (d *ScriptDebugger) RunScript(ctx context.Context) (value cadence.Value, txErr, processError error) {
//
// 	client, err := getClient(d.archiveHost, d.log)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	defer func() {
// 		err := client.Close()
// 		if err != nil {
// 			d.log.Warn().
// 				Err(err).
// 				Msg("Could not close client connection.")
// 		}
// 	}()
//
// 	// last, err := client.GetLast(ctx, &archive.GetLastRequest{})
// 	// if err != nil {
// 	// 	return nil, nil, err
// 	// }
// 	// blockHeight := last.Height
//
// 	blockHeight := d.blockHeight
//
// 	readFunc := func(address string, key string) (flow.RegisterValue, error) {
// 		ledgerKey := state.RegisterIDToKey(flow.RegisterID{Key: key, Owner: address})
// 		ledgerPath, err := pathfinder.KeyToPath(ledgerKey, complete.DefaultPathFinderVersion)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		resp, err := client.GetRegisterValues(ctx, &archive.GetRegisterValuesRequest{
// 			Height: blockHeight,
// 			Paths:  [][]byte{ledgerPath[:]},
// 		})
// 		if err != nil {
// 			return nil, err
// 		}
// 		return resp.Values[0], nil
// 	}
//
// 	cache, err := registers.NewRemoteRegisterFileCache(blockHeight, d.log)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	d.registerGetWrappers = []registers.RegisterGetWrapper{
// 		cache,
// 		registers.NewRemoteRegisterReadTracker(d.directory, d.log),
// 		registers.NewCaptureContractWrapper(d.directory, d.log),
// 	}
//
// 	for _, wrapper := range d.registerGetWrappers {
// 		readFunc = wrapper.Wrap(readFunc)
// 	}
//
// 	view := NewRemoteView(readFunc)
//
// 	d.logHandlers = []LogHandler{NewIntensitiesLogHandler(d.log, d.directory)}
//
// 	d.debugger = NewRemoteDebugger(view, d.chain, d.directory, d.log.Output(
// 		&LogInterceptor{
// 			handlers: d.logHandlers,
// 		}))
//
// 	err = d.dumpScriptToFile(d.script)
//
// 	value, scriptErr, err := d.debugger.RunScript(d.script)
//
// 	_ = d.cleanup()
//
// 	return value, scriptErr, err
// }
//
// func (d *ScriptDebugger) cleanup() error {
// 	var result *multierror.Error
//
// 	// close all log interceptors
// 	for _, handler := range d.logHandlers {
// 		switch w := handler.(type) {
// 		case io.Closer:
// 			result = multierror.Append(result, w.Close())
// 		}
// 	}
//
// 	// close all the register read wrappers
// 	for _, wrapper := range d.registerGetWrappers {
// 		switch w := wrapper.(type) {
// 		case io.Closer:
// 			result = multierror.Append(result, w.Close())
// 		}
// 	}
// 	return result.ErrorOrNil()
// }
//
// func (d *ScriptDebugger) dumpScriptToFile(script *fvm.ScriptProcedure) error {
// 	filename := d.directory + "/script.cdc"
// 	err := os.MkdirAll(filepath.Dir(filename), os.ModePerm)
// 	if err != nil {
// 		return err
// 	}
// 	file, err := os.Create(filename)
// 	if err != nil {
// 		return err
// 	}
// 	defer func(csvFile *os.File) {
// 		err := csvFile.Close()
// 		if err != nil {
// 			d.log.Warn().
// 				Err(err).
// 				Msg("Could not close file.")
// 		}
// 	}(file)
//
// 	_, err = file.WriteString(string(script.Script))
// 	return err
//
// }
