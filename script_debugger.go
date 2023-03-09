package main

import (
	"context"
	"github.com/hashicorp/go-multierror"
	"github.com/janezpodhostnik/flow-transaction-info/registers"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/execution"
	"github.com/rs/zerolog"
	"io"
	"os"
	"path/filepath"
)

type ScriptDebugger struct {
	script *fvm.ScriptProcedure
	// blockHeight uint64
	archiveHost string
	chain       flow.Chain

	directory string

	log zerolog.Logger

	registerGetWrappers []registers.RegisterGetWrapper

	debugger    *RemoteDebugger
	logHandlers []LogHandler
}

func NewScriptDebugger(
	script *fvm.ScriptProcedure,
	// blockHeight uint64,
	archiveHost string,
	chain flow.Chain,
	logger zerolog.Logger) *ScriptDebugger {

	return &ScriptDebugger{
		script: script,
		// blockHeight: blockHeight,
		archiveHost: archiveHost,
		chain:       chain,

		directory: "s_" + script.ID.String(),

		log: logger,
	}
}

func (d *ScriptDebugger) RunScript(ctx context.Context) (value cadence.Value, txErr, processError error) {

	client, err := getExeClient(d.archiveHost, d.log)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		err := client.Close()
		if err != nil {
			d.log.Warn().
				Err(err).
				Msg("Could not close client connection.")
		}
	}()

	header, err := client.GetLatestBlockHeader(ctx, &execution.GetLatestBlockHeaderRequest{})
	if err != nil {
		return nil, nil, err
	}
	blockId := header.Block.Id

	readFunc := func(address string, key string) (flow.RegisterValue, error) {

		resp, err := client.GetRegisterAtBlockID(ctx, &execution.GetRegisterAtBlockIDRequest{
			BlockId:       blockId,
			RegisterOwner: []byte(address),
			RegisterKey:   []byte(key),
		})
		if err != nil {
			d.log.Warn().Err(err).Msg("Could not get register value.")
			return nil, err
		}

		return resp.Value, nil
	}

	cache, err := registers.NewRemoteRegisterFileCache(header.Block.Height, d.log)
	if err != nil {
		return nil, nil, err
	}
	d.registerGetWrappers = []registers.RegisterGetWrapper{
		cache,
		registers.NewRemoteRegisterReadTracker(d.directory, d.log),
		registers.NewCaptureContractWrapper(d.directory, d.log),
	}

	for _, wrapper := range d.registerGetWrappers {
		readFunc = wrapper.Wrap(readFunc)
	}

	view := NewRemoteView(readFunc)

	d.logHandlers = []LogHandler{NewIntensitiesLogHandler(d.log, d.directory)}

	d.debugger = NewRemoteDebugger(view, d.chain, d.directory, d.log.Output(
		&LogInterceptor{
			handlers: d.logHandlers,
		}))

	err = d.dumpScriptToFile(d.script)

	value, scriptErr, err := d.debugger.RunScript(d.script)

	_ = d.cleanup()

	return value, scriptErr, err
}

func (d *ScriptDebugger) cleanup() error {
	var result *multierror.Error

	err := d.debugger.Close()
	if err != nil {
		result = multierror.Append(result, err)
	}

	// close all log interceptors
	for _, handler := range d.logHandlers {
		switch w := handler.(type) {
		case io.Closer:
			result = multierror.Append(result, w.Close())
		}
	}

	// close all the register read wrappers
	for _, wrapper := range d.registerGetWrappers {
		switch w := wrapper.(type) {
		case io.Closer:
			result = multierror.Append(result, w.Close())
		}
	}
	return result.ErrorOrNil()
}

func (d *ScriptDebugger) dumpScriptToFile(script *fvm.ScriptProcedure) error {
	filename := d.directory + "/script.cdc"
	err := os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func(csvFile *os.File) {
		err := csvFile.Close()
		if err != nil {
			d.log.Warn().
				Err(err).
				Msg("Could not close file.")
		}
	}(file)

	_, err = file.WriteString(string(script.Script))
	return err

}