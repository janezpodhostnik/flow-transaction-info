package main

import (
	"context"
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/onflow/flow-go/model/flow"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	host := ""

	ctx := context.Background()
	chainID := flow.Testnet

	var remoteData RemoteData

	remoteData, err := NewExecutionDataClient(host, log.Logger)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Could not connect to execution node.")
		return
	}

	defer func(remoteData RemoteData) {
		err := remoteData.Close()
		if err != nil {
			log.Error().
				Err(err).
				Msg("Could not close remote data.")
		}
	}(remoteData)

	txID, _ := flow.HexStringToIdentifier("981fd6a429fdc931fa24b565da5ccf026cbd28a91cd2c61679f70aa40edfe017")
	blockId, _ := flow.HexStringToIdentifier("c690fd56d45b350c2e26c3925dc4b3a3d7fe46c07d403e618553963842450aad")

	txDebugger := NewTransactionDebugger(txID, remoteData, chainID.Chain(), log.Logger)

	tx :=
		flow.NewTransactionBody().
			SetScript([]byte(`
import FLOAT from 0x0afe396ebc8eee65
import NonFungibleToken from 0x631e88ae7f1d7c20
import MetadataViews from 0x631e88ae7f1d7c20
import GrantedAccountAccess from 0x0afe396ebc8eee65
import ChildAccount from 0x1b655847a90e644a

transaction(
  pubKey: String,
  fundingAmt: UFix64,
  childAccountName: String,
  childAccountDescription: String,
  clientIconURL: String,
  clientExternalURL: String
) {

  prepare(signer: AuthAccount) {
    // Save a ChildAccountCreator if none exists
    if signer.borrow<&ChildAccount.ChildAccountCreator>(from: ChildAccount.ChildAccountCreatorStoragePath) == nil {
      signer.save(<-ChildAccount.createChildAccountCreator(), to: ChildAccount.ChildAccountCreatorStoragePath)
    }
    // Link the public Capability so signer can query address on public key
    if !signer.getCapability<
        &ChildAccount.ChildAccountCreator{ChildAccount.ChildAccountCreatorPublic}
      >(ChildAccount.ChildAccountCreatorPublicPath).check() {
      // Unlink & Link
      signer.unlink(ChildAccount.ChildAccountCreatorPublicPath)
      signer.link<
        &ChildAccount.ChildAccountCreator{ChildAccount.ChildAccountCreatorPublic}
      >(
        ChildAccount.ChildAccountCreatorPublicPath,
        target: ChildAccount.ChildAccountCreatorStoragePath
      )
    }
    // Get a reference to the ChildAccountCreator
    let creatorRef = signer.borrow<&ChildAccount.ChildAccountCreator>(
        from: ChildAccount.ChildAccountCreatorStoragePath
      ) ?? panic("Problem getting a ChildAccountCreator reference!")
    // Construct the ChildAccountInfo metadata struct
    let info = ChildAccount.ChildAccountInfo(
        name: childAccountName,
        description: childAccountDescription,
        clientIconURL: MetadataViews.HTTPFile(url: clientIconURL),
        clienExternalURL: MetadataViews.ExternalURL(clientExternalURL),
        originatingPublicKey: pubKey
      )
    // Create the account, passing signer AuthAccount to fund account creation
    // and add initialFundingAmount in Flow if desired
    let newAccount: AuthAccount = creatorRef.createChildAccount(
        signer: signer,
        initialFundingAmount: fundingAmt,
        childAccountInfo: info
      )
    // At this point, the newAccount can further be configured as suitable for
    // use in your dApp (e.g. Setup a Collection, Mint NFT, Configure Vault, etc.)
    // ...


    // SETUP COLLECTION
    if newAccount.borrow<&FLOAT.Collection>(from: FLOAT.FLOATCollectionStoragePath) == nil {
      newAccount.save(<- FLOAT.createEmptyCollection(), to: FLOAT.FLOATCollectionStoragePath)
      newAccount.link<&FLOAT.Collection{NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic, MetadataViews.ResolverCollection, FLOAT.CollectionPublic}>
                (FLOAT.FLOATCollectionPublicPath, target: FLOAT.FLOATCollectionStoragePath)
    }

    // SETUP FLOATEVENTS
    if newAccount.borrow<&FLOAT.FLOATEvents>(from: FLOAT.FLOATEventsStoragePath) == nil {
      newAccount.save(<- FLOAT.createEmptyFLOATEventCollection(), to: FLOAT.FLOATEventsStoragePath)
      newAccount.link<&FLOAT.FLOATEvents{FLOAT.FLOATEventsPublic, MetadataViews.ResolverCollection}>
                (FLOAT.FLOATEventsPublicPath, target: FLOAT.FLOATEventsStoragePath)
    }

    // SETUP SHARED MINTING
    if newAccount.borrow<&GrantedAccountAccess.Info>(from: GrantedAccountAccess.InfoStoragePath) == nil {
      newAccount.save(<- GrantedAccountAccess.createInfo(), to: GrantedAccountAccess.InfoStoragePath)
      newAccount.link<&GrantedAccountAccess.Info{GrantedAccountAccess.InfoPublic}>
                (GrantedAccountAccess.InfoPublicPath, target: GrantedAccountAccess.InfoStoragePath)
    }
  }

  execute {
    log("Finished setting up the account for FLOATs.")
  }
}
`)).
			AddArgument(json.MustEncode(cadence.String("baa22071032f5c48fc6cde5563334f9d877fd52e568ec9307cb8bceca926f2343d12479777111990edb508f886ffae1d1ac30c8ab8983370cc210d771ded1524"))).
			AddArgument(json.MustEncode(func() cadence.UFix64 { v, _ := cadence.NewUFix64("0.10000000"); return v }())).
			AddArgument(json.MustEncode(cadence.String("PayGlide Proxy Account"))).
			AddArgument(json.MustEncode(cadence.String(""))).
			AddArgument(json.MustEncode(cadence.String("demo.payglide.xyz/payglide.png"))).
			AddArgument(json.MustEncode(cadence.String("demo.payglide.xyz"))).
			AddAuthorizer(flow.HexToAddress("0x96850d70856f80d4")).
			SetProposalKey(flow.HexToAddress("0x96850d70856f80d4"), 0, 0)

	txErr, err := txDebugger.RunTransaction(ctx, blockId, tx)

	if txErr != nil {
		log.Error().
			Err(txErr).
			Msg("Transaction error.")
		return
	}
	if err != nil {
		log.Error().
			Err(err).
			Msg("Implementation error.")
		return
	}
}
