package main

import (
	"context"
	"flag"
	"github.com/onflow/flow-dps/api/dps"
	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var host string
	flag.StringVar(&host, "host", "dps-001.mainnet20.nodes.onflow.org:9000", "host url with port")
	flag.Parse()

	txid, err := flow.HexStringToIdentifier("xxx")
	if err != nil {
		log.Error().
			Err(err).
			Msg("Could not parse transaction ID.")
		return
	}

	chain := flow.Mainnet.Chain()
	ctx := context.Background()

	conn, err := grpc.Dial(host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("host", host).
			Msg("Could not connect to server.")
		return
	}
	client := dps.NewAPIClient(conn)

	resp, err := client.GetHeightForTransaction(ctx, &dps.GetHeightForTransactionRequest{
		TransactionID: txid[:],
	})
	if err != nil {
		log.Error().
			Err(err).
			Msg("Could not get transaction block height.")
		return
	}
	blockHeight := resp.GetHeight()

	log.Info().
		Uint64("height", blockHeight).
		Msg("Running transaction.")

	readFunc := func(address string, key string) (flow.RegisterValue, error) {
		ledgerKey := state.RegisterIDToKey(flow.RegisterID{Key: key, Owner: address})
		ledgerPath, err := pathfinder.KeyToPath(ledgerKey, complete.DefaultPathFinderVersion)
		if err != nil {
			return nil, err
		}

		resp, err := client.GetRegisterValues(ctx, &dps.GetRegisterValuesRequest{
			Height: blockHeight,
			Paths:  [][]byte{ledgerPath[:]},
		})
		if err != nil {
			return nil, err
		}
		return resp.Values[0], nil
	}

	readCache, err := NewRemoteRegisterFileCache(readFunc, blockHeight, log.Logger)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Could not create register cache.")
		return
	}
	defer func() {
		err := readCache.Close()
		if err != nil {
			log.Warn().
				Err(err).
				Msg("Could not close register cache.")
		}
	}()

	tracker := NewRemoteRegisterReadTracker(readCache.Get, log.Logger)
	defer tracker.LogReads()

	view := NewRemoteView(tracker.Get)

	debugger := NewRemoteDebugger(view, chain, log.Logger)

	codec := zbor.NewCodec()

	txResult, err := client.GetTransaction(ctx, &dps.GetTransactionRequest{
		TransactionID: txid[:],
	})
	if err != nil {
		log.Error().
			Err(err).
			Msg("Could not get transaction.")
		return
	}
	var txBody flow.TransactionBody
	err = codec.Unmarshal(txResult.Data, &txBody)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Could not unmarshal transaction.")
		return
	}

	txErr, err := debugger.RunTransaction(&txBody)
	if txErr != nil {
		log.Error().
			Err(txErr).
			Msg("Transaction error.")
		return
	}
	if err != nil {
		log.Error().
			Err(txErr).
			Msg("Implementation error.")
		return
	}
}
