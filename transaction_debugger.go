package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"github.com/hashicorp/go-multierror"
	"github.com/janezpodhostnik/flow-transaction-info/registers"
	"github.com/onflow/flow-archive/api/archive"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/execution"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type TransactionDebugger struct {
	txID       flow.Identifier
	remoteData RemoteData
	chain      flow.Chain

	directory string

	log zerolog.Logger

	registerGetWrappers []registers.RegisterGetWrapper

	debugger    *RemoteDebugger
	logHandlers []LogHandler
}

func NewTransactionDebugger(
	txID flow.Identifier,
	remoteData RemoteData,
	chain flow.Chain,
	logger zerolog.Logger) *TransactionDebugger {

	return &TransactionDebugger{
		txID:       txID,
		remoteData: remoteData,
		chain:      chain,

		directory: "t_" + txID.String(),

		log: logger,
	}
}

type clientWithConnection struct {
	archive.APIClient
	*grpc.ClientConn
}

func (d *TransactionDebugger) RunTransaction(ctx context.Context, blockID flow.Identifier, txBody *flow.TransactionBody) (txErr, processError error) {
	d.log.Info().
		Str("txID", d.txID.String()).
		Msg("Re-running transaction. This may differ from how the transaction was originally run on the network," +
			" due to the fact that some steps are skipped (signature verification, sequence number verification, ...)" +
			" and that the transaction is run on the state as it at the beginning of the block " +
			"(which might not have ben the case originally).")

	cache, err := registers.NewRemoteRegisterFileCache(blockID.String(), d.log)
	if err != nil {
		return nil, err
	}
	d.registerGetWrappers = []registers.RegisterGetWrapper{
		cache,
		registers.NewRemoteRegisterReadTracker(d.directory, d.log),
		registers.NewCaptureContractWrapper(d.directory, d.log),
	}

	readFunc := func(owner string, key string) (flow.RegisterValue, error) {
		return d.remoteData.GetRemoteRegister(ctx, blockID, owner, key)
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

	// txBody, err := d.remoteData.GetTransaction(ctx, d.txID)
	// if err != nil {
	// 	return nil, err
	// }
	//
	// err = d.dumpTransactionToFile(txBody)
	// if err != nil {
	// 	return nil, err
	// }

	txErr, err = d.debugger.RunTransaction(txBody)

	_ = d.cleanup()

	return txErr, err
}

func (d *TransactionDebugger) cleanup() error {
	var result *multierror.Error

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

// getClient returns a client to the Archive API.
func getExeClient(archiveHost string, log zerolog.Logger) (exeClientWithConnection, error) {
	conn, err := grpc.Dial(
		archiveHost,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("host", archiveHost).
			Msg("Could not connect to server.")
		return exeClientWithConnection{}, err
	}
	client := execution.NewExecutionAPIClient(conn)

	return exeClientWithConnection{
		ExecutionAPIClient: client,
		ClientConn:         conn,
	}, nil
}

type exeClientWithConnection struct {
	execution.ExecutionAPIClient
	*grpc.ClientConn
}

func (d *TransactionDebugger) dumpTransactionToFile(body flow.TransactionBody) error {
	filename := d.directory + "/transaction.cdc"
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

	_, err = file.WriteString(string(body.Script))
	return err
}

type LogHandler interface {
	Handle(string) error
}

type IntensitiesLogHandler struct {
	ComputationIntensities map[uint64]uint64 `json:"computationIntensities"`
	MemoryIntensities      map[uint64]uint64 `json:"memoryIntensities"`

	log      zerolog.Logger
	filename string
}

type LogInterceptor struct {
	handlers []LogHandler
}

func NewIntensitiesLogHandler(log zerolog.Logger, directory string) *IntensitiesLogHandler {
	return &IntensitiesLogHandler{
		ComputationIntensities: map[uint64]uint64{},
		MemoryIntensities:      map[uint64]uint64{},
		log:                    log,
		filename:               directory + "/computation_intensities.csv",
	}
}

var _ io.Writer = &LogInterceptor{}

type computationIntensitiesLog struct {
	ComputationIntensities map[uint64]uint64 `json:"computationIntensities"`
	MemoryIntensities      map[uint64]uint64 `json:"memoryIntensities"`
}

func (l *LogInterceptor) Write(p []byte) (n int, err error) {
	for _, handler := range l.handlers {
		err := handler.Handle(string(p))
		if err != nil {
			return 0, err
		}
	}
	return len(p), nil
}
func (l *IntensitiesLogHandler) Handle(line string) error {
	if strings.Contains(line, "computationIntensities") {
		var log computationIntensitiesLog
		err := json.Unmarshal([]byte(line), &log)
		if err != nil {
			return err
		}
		l.ComputationIntensities = log.ComputationIntensities
		l.MemoryIntensities = log.MemoryIntensities
	}
	return nil
}

func (l *IntensitiesLogHandler) Close() error {
	err := os.MkdirAll(filepath.Dir(l.filename), os.ModePerm)
	if err != nil {
		return err
	}
	csvFile, err := os.Create(l.filename)
	if err != nil {
		return err
	}
	defer func(csvFile *os.File) {
		err := csvFile.Close()
		if err != nil {
			l.log.Warn().
				Err(err).
				Msg("Could not close csv file.")
		}
	}(csvFile)

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()
	err = writer.Write([]string{"*Computation Kind", "Intensity"})
	if err != nil {
		return err
	}
	for i, q := range l.ComputationIntensities {
		key, ok := computationKindNameMap[i]
		if !ok {
			key = strconv.Itoa(int(i))
		}

		err := writer.Write([]string{key, strconv.Itoa(int(q))})
		if err != nil {
			return err
		}
	}

	return nil
}

var computationKindNameMap = map[uint64]string{
	1001: "*Statement",
	1002: "*Loop",
	1003: "*FunctionInvocation",
	1010: "CreateCompositeValue",
	1011: "TransferCompositeValue",
	1012: "DestroyCompositeValue",
	1025: "CreateArrayValue",
	1026: "TransferArrayValue",
	1027: "DestroyArrayValue",
	1040: "CreateDictionaryValue",
	1041: "TransferDictionaryValue",
	1042: "DestroyDictionaryValue",
	1100: "STDLIBPanic",
	1101: "STDLIBAssert",
	1102: "STDLIBUnsafeRandom",
	1108: "STDLIBRLPDecodeString",
	1109: "STDLIBRLPDecodeList",
	2001: "Hash",
	2002: "VerifySignature",
	2003: "AddAccountKey",
	2004: "AddEncodedAccountKey",
	2005: "AllocateStorageIndex",
	2006: "*CreateAccount",
	2007: "EmitEvent",
	2008: "GenerateUUID",
	2009: "GetAccountAvailableBalance",
	2010: "GetAccountBalance",
	2011: "GetAccountContractCode",
	2012: "GetAccountContractNames",
	2013: "GetAccountKey",
	2014: "GetBlockAtHeight",
	2015: "GetCode",
	2016: "GetCurrentBlockHeight",
	2017: "GetProgram",
	2018: "GetStorageCapacity",
	2019: "GetStorageUsed",
	2020: "*GetValue",
	2021: "RemoveAccountContractCode",
	2022: "ResolveLocation",
	2023: "RevokeAccountKey",
	2034: "RevokeEncodedAccountKey",
	2025: "SetProgram",
	2026: "*SetValue",
	2027: "UpdateAccountContractCode",
	2028: "ValidatePublicKey",
	2029: "ValueExists",
}
