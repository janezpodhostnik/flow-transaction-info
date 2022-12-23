package main

import (
	"context"
	"flag"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var host string
	flag.StringVar(&host, "host", "", "host url with port")

	var tx string
	flag.StringVar(&tx, "tx", "", "transaction id")

	flag.Parse()

	txid, err := flow.HexStringToIdentifier(tx)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Could not parse transaction ID.")
		return
	}

	chain := flow.Mainnet.Chain()
	ctx := context.Background()

	txErr, err := NewTransactionDebugger(txid, host, chain, log.Logger).RunTransaction(ctx)

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
