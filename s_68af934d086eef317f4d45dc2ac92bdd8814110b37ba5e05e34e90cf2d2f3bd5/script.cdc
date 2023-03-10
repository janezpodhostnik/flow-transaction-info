
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
        NFTCatalog.getCatalogEntry(collectionIdentifier: collectionIdentifier) != nil : "Invalid collection identifier"
    }

    let value = NFTCatalog.getCatalogEntry(collectionIdentifier: collectionIdentifier)!
    let items: {String: AnyStruct} = {}

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

pub fun getNFTViewsFromIDs(collectionIdentifier : String, ids: [UInt64], collectionCap : Capability<&AnyResource{MetadataViews.ResolverCollection}>) : [MetadataViews.NFTView] {
        pre {
            NFTCatalog.getCatalogEntry(collectionIdentifier: collectionIdentifier) != nil : "Invalid collection identifier"
        }

        let value = NFTCatalog.getCatalogEntry(collectionIdentifier: collectionIdentifier)!
        let items : [MetadataViews.NFTView] = []

        // Check if we have multiple collections for the NFT type...
        let hasMultipleCollections = false

         if collectionCap.check() {
            let collectionRef = collectionCap.borrow()!
            for id in ids {
                if !collectionRef.getIDs().contains(id) {
                    continue
                }
                let nftResolver = collectionRef.borrowViewResolver(id: id)
                let nftViews = MetadataViews.getNFTView(id: id, viewResolver: nftResolver)
                if !hasMultipleCollections {
                    items.append(nftViews)
                } else if nftViews.display!.name == value.collectionDisplay.name {
                    items.append(nftViews)
                }
            
            }
        }


        return items
    }

pub fun main(ownerAddress: Address, collectionIdentifier: String, tokenID: UInt64) : NFT? {
    let account = getAuthAccount(ownerAddress)

    let value = NFTCatalog.getCatalogEntry(collectionIdentifier: collectionIdentifier)!
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

    let views = getNFTViewsFromIDs(collectionIdentifier: collectionIdentifier,ids: [], collectionCap: collectionCap)

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
}