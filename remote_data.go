package main

import (
	"context"
	"github.com/onflow/flow-archive/api/archive"
	"github.com/onflow/flow-archive/codec/zbor"
	"github.com/onflow/flow/protobuf/go/flow/execution"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"

	"github.com/onflow/flow-go/model/flow"
)

type RemoteData interface {
	io.Closer
	GetRemoteRegister(ctx context.Context, blockId flow.Identifier, owner, key string) (flow.RegisterValue, error)
}

type ArchiveDataClient struct {
	client clientWithConnection
	log    zerolog.Logger
}

func NewArchiveDataClient(archiveHost string, log zerolog.Logger) (*ArchiveDataClient, error) {
	conn, err := grpc.Dial(
		archiveHost,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("host", archiveHost).
			Msg("Could not connect to server.")
		return nil, err
	}

	client := clientWithConnection{
		APIClient:  archive.NewAPIClient(conn),
		ClientConn: conn,
	}

	return &ArchiveDataClient{
		client: client,
		log:    log,
	}, nil
}

func (a *ArchiveDataClient) Close() error {
	return a.client.Close()
}

func (a *ArchiveDataClient) GetRemoteRegister(ctx context.Context, blockId flow.Identifier, address, key string) (flow.RegisterValue, error) {
	panic("why cant I get block height from block ID using the archive API? ;(")

	// ledgerKey := state.RegisterIDToKey(flow.RegisterID{
	// 	Key:   key,
	// 	Owner: address,
	// })
	//
	// ledgerPath, err := pathfinder.KeyToPath(ledgerKey, complete.DefaultPathFinderVersion)
	// if err != nil {
	// 	return nil, err
	// }
	//
	// resp, err := a.client.GetRegisterValues(ctx, &archive.GetRegisterValuesRequest{
	// 	Height: blockHeight,
	// 	Paths:  [][]byte{ledgerPath[:]},
	// })
	// if err != nil {
	// 	return nil, err
	// }
	// return resp.Values[0], nil
}

func (a *ArchiveDataClient) GetTransactionBlockHeight(ctx context.Context, txID flow.Identifier) (uint64, error) {
	resp, err := a.client.GetHeightForTransaction(ctx, &archive.GetHeightForTransactionRequest{
		TransactionID: txID[:],
	})
	if err != nil {
		a.log.Error().
			Err(err).
			Msg("Could not get transaction block height.")
		return 0, err
	}
	blockHeight := resp.GetHeight()

	a.log.Info().
		Uint64("height", blockHeight).
		Msg("Got block height for transaction.")
	return blockHeight, nil
}

func (a *ArchiveDataClient) GetTransaction(ctx context.Context, txID flow.Identifier) (flow.TransactionBody, error) {
	codec := zbor.NewCodec()

	txResult, err := a.client.GetTransaction(ctx, &archive.GetTransactionRequest{
		TransactionID: txID[:],
	})
	if err != nil {
		a.log.Error().
			Err(err).
			Msg("Could not get transaction.")
		return flow.TransactionBody{}, err
	}
	var txBody flow.TransactionBody
	err = codec.Unmarshal(txResult.Data, &txBody)
	if err != nil {
		a.log.Error().
			Err(err).
			Msg("Could not unmarshal transaction.")
		return flow.TransactionBody{}, err
	}
	return txBody, nil
}

var _ RemoteData = (*ArchiveDataClient)(nil)

type ExecutionDataClient struct {
	client  exeClientWithConnection
	log     zerolog.Logger
	blockID map[uint64]flow.Identifier
}

func (e *ExecutionDataClient) Close() error {
	return e.client.Close()
}

func (e *ExecutionDataClient) GetRemoteRegister(ctx context.Context, blockId flow.Identifier, owner, key string) (flow.RegisterValue, error) {

	req := &execution.GetRegisterAtBlockIDRequest{
		BlockId:       blockId[:],
		RegisterOwner: []byte(owner),
		RegisterKey:   []byte(key),
	}

	resp, err := e.client.GetRegisterAtBlockID(
		ctx,
		req)
	if err != nil {
		return nil, err
	}

	return resp.Value, nil
}

func (e *ExecutionDataClient) GetTransaction(ctx context.Context, txID flow.Identifier) (flow.TransactionBody, error) {
	// TODO implement me
	panic("implement me")
}

func NewExecutionDataClient(executionHost string, log zerolog.Logger) (*ExecutionDataClient, error) {
	conn, err := grpc.Dial(
		executionHost,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("host", executionHost).
			Msg("Could not connect to server.")
		return nil, err
	}
	client := exeClientWithConnection{
		ExecutionAPIClient: execution.NewExecutionAPIClient(conn),
		ClientConn:         conn,
	}

	return &ExecutionDataClient{
		client: client,
		log:    log,
	}, nil
}

var _ RemoteData = (*ExecutionDataClient)(nil)
