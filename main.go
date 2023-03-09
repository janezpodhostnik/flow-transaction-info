package main

import (
	"context"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"

	jsoncdc "github.com/onflow/cadence/encoding/json"
)

//
// func main() {
// 	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
//
// 	var host string
// 	flag.StringVar(&host, "host", "", "host url with port")
//
// 	var tx string
// 	flag.StringVar(&tx, "tx", "", "transaction id")
//
// 	var chainStr string
// 	flag.StringVar(&chainStr, "chain", "flow-mainnet", "chain id (flow-mainnet, flow-testnet, ...)")
//
// 	flag.Parse()
//
// 	txid, err := flow.HexStringToIdentifier(tx)
// 	if err != nil {
// 		log.Error().
// 			Err(err).
// 			Msg("Could not parse transaction ID.")
// 		return
// 	}
//
// 	chainID := flow.ChainID(chainStr)
// 	ctx := context.Background()
//
// 	txDebugger := NewTransactionDebugger(txid, host, chainID.Chain(), log.Logger)
//
// 	txErr, err := txDebugger.RunTransaction(ctx)
//
// 	if txErr != nil {
// 		log.Error().
// 			Err(txErr).
// 			Msg("Transaction error.")
// 		return
// 	}
// 	if err != nil {
// 		log.Error().
// 			Err(err).
// 			Msg("Implementation error.")
// 		return
// 	}
// }

func main() {
	scriptCode := `
import MetadataViews from 0x1d7e57aa55817448
import NFTCatalog from 0x49a7cda3a1eecc29
import NFTRetrieval from 0x49a7cda3a1eecc29

pub struct NFTCollectionData {
    pub let storagePath: StoragePath
    pub let publicPath: PublicPath
    pub let privatePath: PrivatePath
    pub let publicLinkedType: Type
    pub let privateLinkedType: Type

    init(
        storagePath: StoragePath,
        publicPath: PublicPath,
        privatePath: PrivatePath,
        publicLinkedType: Type,
        privateLinkedType: Type,
    ) {
        self.storagePath = storagePath
        self.publicPath = publicPath
        self.privatePath = privatePath
        self.publicLinkedType = publicLinkedType
        self.privateLinkedType = privateLinkedType
    }
}

pub struct NFT {
    pub let id: UInt64
    pub let uuid : UInt64
    pub let name: String
    pub let description: String
    pub let thumbnail: String
    pub let externalURL: String
    pub let storagePath: StoragePath
    pub let publicPath: PublicPath
    pub let privatePath: PrivatePath
    pub let publicLinkedType: Type
    pub let privateLinkedType: Type
    pub let collectionName: String
    pub let collectionDescription: String
    pub let collectionSquareImage: String
    pub let collectionBannerImage: String
    pub let collectionExternalURL: String
    pub let allViews:  {String: AnyStruct}?
    pub let royalties: [MetadataViews.Royalty]
    pub let collectionSocials: {String : MetadataViews.ExternalURL}
    pub let traits : MetadataViews.Traits?

    init(
        id: UInt64,
        uuid : UInt64,
        name: String,
        description: String,
        thumbnail: String,
        externalURL: String,
        storagePath: StoragePath,
        publicPath: PublicPath,
        privatePath: PrivatePath,
        publicLinkedType: Type,
        privateLinkedType: Type,
        collectionName: String,
        collectionDescription: String,
        collectionSquareImage: String,
        collectionBannerImage: String,
        collectionExternalURL: String,
        allViews: {String: AnyStruct}?,
        royalties: [MetadataViews.Royalty],
        collectionSocials: {String: MetadataViews.ExternalURL},
        traits : MetadataViews.Traits?
    ) {
        self.id = id
        self.uuid = uuid
        self.name = name
        self.description = description
        self.thumbnail = thumbnail
        self.externalURL = externalURL
        self.storagePath = storagePath
        self.publicPath = publicPath
        self.privatePath = privatePath
        self.publicLinkedType = publicLinkedType
        self.privateLinkedType = privateLinkedType
        self.collectionName = collectionName
        self.collectionDescription = collectionDescription
        self.collectionSquareImage = collectionSquareImage
        self.collectionBannerImage = collectionBannerImage
        self.collectionExternalURL = collectionExternalURL
        self.allViews = allViews
        self.royalties = royalties
        self.collectionSocials = collectionSocials
        self.traits = traits
    }
}

 pub fun getAllMetadataViewsFromCap(tokenID: UInt64, collectionIdentifier: String, collectionCap: Capability<&AnyResource{MetadataViews.ResolverCollection}>): {String: AnyStruct} {
    pre {
        NFTCatalog.getCatalog()[collectionIdentifier] != nil : "Invalid collection identifier"
    }

    let catalog = NFTCatalog.getCatalog()
    let items: {String: AnyStruct} = {}
    let value = catalog[collectionIdentifier]!

    // Check if we have multiple collections for the NFT type...
    let hasMultipleCollections = false

    if collectionCap.check() {
        let collectionRef = collectionCap.borrow()!

        let nftResolver = collectionRef.borrowViewResolver(id: tokenID)
        let supportedNftViewTypes = nftResolver.getViews()

        for supportedViewType in supportedNftViewTypes {
            if let view = nftResolver.resolveView(supportedViewType) {
                if !hasMultipleCollections {
                    items.insert(key: supportedViewType.identifier, view)
                } else if MetadataViews.getDisplay(nftResolver)!.name == value.collectionDisplay.name {
                    items.insert(key: supportedViewType.identifier, view)
                }
            }
        }

    }

    return items
}

pub fun main(ownerAddress: Address, collectionIdentifier: String, tokenID: UInt64) : NFT? {
    let catalog = NFTCatalog.getCatalog()

    assert(catalog.containsKey(collectionIdentifier), message: "Invalid Collection")

    let account = getAuthAccount(ownerAddress)

    let value = catalog[collectionIdentifier]!
    let identifierHash = String.encodeHex(HashAlgorithm.SHA3_256.hash(collectionIdentifier.utf8))
    let tempPathStr = "catalog".concat(identifierHash)
    let tempPublicPath = PublicPath(identifier: tempPathStr)!

    account.link<&{MetadataViews.ResolverCollection}>(
        tempPublicPath,
        target: value.collectionData.storagePath
    )

    let collectionCap = account.getCapability<&AnyResource{MetadataViews.ResolverCollection}>(tempPublicPath)
    assert(collectionCap.check(), message: "MetadataViews Collection is not set up properly, ensure the Capability was created/linked correctly.")

    let allViews = getAllMetadataViewsFromCap(tokenID: tokenID, collectionIdentifier: collectionIdentifier, collectionCap: collectionCap)
    let nftCollectionDisplayView = allViews[Type<MetadataViews.NFTCollectionData>().identifier] as! MetadataViews.NFTCollectionData?
    let collectionDataView = NFTCollectionData(
        storagePath: nftCollectionDisplayView!.storagePath,
        publicPath: nftCollectionDisplayView!.publicPath,
        privatePath: nftCollectionDisplayView!.providerPath,
        publicLinkedType: nftCollectionDisplayView!.publicLinkedType,
        privateLinkedType: nftCollectionDisplayView!.providerLinkedType,
    )

    allViews.insert(key: Type<MetadataViews.NFTCollectionData>().identifier, collectionDataView)

    let views = NFTRetrieval.getNFTViewsFromCap(collectionIdentifier: collectionIdentifier, collectionCap: collectionCap)

    for view in views {
        if view.id == tokenID {
            let displayView = view.display
            let externalURLView = view.externalURL
            let collectionDataView = view.collectionData
            let collectionDisplayView = view.collectionDisplay
            let royaltyView = view.royalties

            if (displayView == nil || externalURLView == nil || collectionDataView == nil || collectionDisplayView == nil || royaltyView == nil) {
                // Bad NFT. Skipping....
                return nil
            }

            return NFT(
                id: view.id,
                uuid: view.uuid,
                name: displayView!.name,
                description: displayView!.description,
                thumbnail: displayView!.thumbnail.uri(),
                externalURL: externalURLView!.url,
                storagePath: collectionDataView!.storagePath,
                publicPath: collectionDataView!.publicPath,
                privatePath: collectionDataView!.providerPath,
                publicLinkedType: collectionDataView!.publicLinkedType,
                privateLinkedType: collectionDataView!.providerLinkedType,
                collectionName: collectionDisplayView!.name,
                collectionDescription: collectionDisplayView!.description,
                collectionSquareImage: collectionDisplayView!.squareImage.file.uri(),
                collectionBannerImage: collectionDisplayView!.bannerImage.file.uri(),
                collectionExternalURL: collectionDisplayView!.externalURL.url,
                allViews: allViews,
                royalties: royaltyView!.getRoyalties(),
                collectionSocials: collectionDisplayView!.socials,
                traits: view.traits
            )
        }
    }

    return nil
}`

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	// blockHeight := uint64(44151459)
	host := "34.68.103.212:9000"
	chain := flow.Testnet.Chain()

	script := fvm.NewScriptWithContextAndArgs(
		[]byte(scriptCode),
		context.Background(),
		jsoncdc.MustEncode(cadence.NewAddress(flow.HexToAddress("0x01cf0e2f2f715450"))),
		jsoncdc.MustEncode(cadence.String("NBATopShot")),
		jsoncdc.MustEncode(cadence.UInt64(38501733)),
	)

	scriptDebugger := NewScriptDebugger(
		script,
		// blockHeight,
		host,
		chain,
		log.Logger)
	ctx := context.Background()
	_, scriptErr, err := scriptDebugger.RunScript(ctx)
	if scriptErr != nil {
		log.Error().
			Err(scriptErr).
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
