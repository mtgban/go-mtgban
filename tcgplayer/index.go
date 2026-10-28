package tcgplayer

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	"github.com/mtgban/go-tcgplayer"
)

// Index prices Magic singles from the partner API's price guide, the
// low and market numbers rather than any one seller's listing.
type Index struct {
	logCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	affiliate      string
	maxConcurrency int

	inventory mtgban.InventoryRecord

	backend *mtgmatcher.Backend
	client  *tcgplayer.Client

	// printings maps a product id and the subtype of its price rows to
	// the printing that product sells in that finish.
	printings map[string]map[string]string
}

var availableIndexNames = []string{
	"TCG Low", "TCG Market", "TCG Mid", "TCG Direct Low",
}

func (tcg *Index) printf(format string, a ...any) {
	if tcg.logCallback != nil {
		tcg.logCallback("[TCGIndex] "+format, a...)
	}
}

// NewScraperIndex returns an index scraper authenticated with a partner API
// key pair.
func NewScraperIndex(b *mtgmatcher.Backend, publicID, privateID string) (*Index, error) {
	client, err := tcgplayer.NewClient(publicID, privateID)
	if err != nil {
		return nil, err
	}

	tcg := Index{}
	tcg.backend = b
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.client = client
	tcg.maxConcurrency = defaultConcurrency
	return &tcg, nil
}

func (tcg *Index) processEntry(ctx context.Context, channel chan<- responseChan, reqs []string) error {
	var ids []int
	for _, req := range reqs {
		id, err := strconv.Atoi(req)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	results, err := tcg.client.GetMarketPricesByProducts(ctx, ids)
	if err != nil {
		return err
	}

	for _, result := range results {
		// Skip empty entries
		if result.LowPrice == 0 && result.MarketPrice == 0 && result.MidPrice == 0 && result.DirectLowPrice == 0 {
			continue
		}

		cardID, found := tcg.printings[fmt.Sprint(result.ProductID)][result.SubTypeName]
		if !found {
			continue
		}
		co, _ := tcg.backend.GetUUID(cardID)

		// These are sorted as in availableIndexNames
		prices := []float64{
			result.LowPrice, result.MarketPrice, result.MidPrice, getDirectPrice(result.DirectLowPrice),
		}

		for i := range availableIndexNames {
			if prices[i] == 0 {
				continue
			}

			// Certain sets are marked as English on the site despite not being as such
			// Override here, so that links don't point at filters that provide no results
			lang := co.Language
			switch co.SetCode {
			case "STA", "SOA":
				lang = "English"
			}

			isDirect := availableIndexNames[i] == "TCG Direct Low"
			link := GenerateProductURL(result.ProductID, result.SubTypeName, tcg.affiliate, "", lang, isDirect)

			out := responseChan{
				cardID: cardID,
				entry: mtgban.InventoryEntry{
					Conditions: mtgban.NM,
					Price:      prices[i],
					Quantity:   1,
					URL:        link,
					SellerName: availableIndexNames[i],
					Bundle:     isDirect,
				},
			}

			channel <- out
		}
	}

	return nil
}

// crossSetProductIDs reports every TCGplayer product id (regular or etched)
// claimed by cards from more than one set, keyed to the sets that claim it.
// A double-faced card's two faces, or a plain misprint beside the copy it
// doubles, always share a set - a same-named oversized sibling does not, and
// neither does an id MTGJSON has simply copied onto the wrong printing (the
// Dungeon of the Mad Mage report this guards against: AFR's ordinary card
// and OAFR's oversized one both carry 245106, which belongs only to the
// oversized product). Spanning more than one set means the id is wrong on
// at least one of them, and pricing from it would be a guess.
func crossSetProductIDs(b *mtgmatcher.Backend) map[string][]string {
	setsByID := map[string]map[string]bool{}
	for _, code := range b.GetAllSets() {
		set, err := b.GetSet(code)
		if err != nil {
			continue
		}
		for _, card := range set.Cards {
			for _, key := range [...]string{"tcgplayerProductId", "tcgplayerEtchedProductId"} {
				id, found := card.Identifiers[key]
				if !found || soldAsCopy(b, id, card.Language) {
					continue
				}
				if setsByID[id] == nil {
					setsByID[id] = map[string]bool{}
				}
				setsByID[id][code] = true
			}
		}
	}

	collisions := map[string][]string{}
	for id, sets := range setsByID {
		if len(sets) <= 1 {
			continue
		}
		for code := range sets {
			collisions[id] = append(collisions[id], code)
		}
		sort.Strings(collisions[id])
	}
	return collisions
}

// soldAsCopy reports whether a printing in language is a copy sold under id,
// a product the datastore files at its original in another language.
func soldAsCopy(b *mtgmatcher.Backend, id, language string) bool {
	original, err := b.GetUUID(b.ConvertID(mtgmatcher.IDSpaceTCGplayer, id))
	return err == nil && original.Language != language
}

// starTwin returns the star printing that is the other finish of the card's
// product, if there is one: the same language, numbered "<number>★" and with
// no product of its own (FRF 65★ is the foil of 95037).
func starTwin(byNumber map[string]*mtgmatcher.CardObject, card *mtgmatcher.Card) *mtgmatcher.CardObject {
	twin, found := byNumber[card.Number+"★|"+card.Language]
	if !found || twin.Identifiers["tcgplayerProductId"] != "" {
		return nil
	}
	return twin
}

// cardsByNumber indexes a set's cards by number and language.
func cardsByNumber(b *mtgmatcher.Backend, set *mtgmatcher.Set) map[string]*mtgmatcher.CardObject {
	byNumber := map[string]*mtgmatcher.CardObject{}
	for _, card := range set.Cards {
		co, found := b.UUIDs[card.UUID]
		if found {
			byNumber[co.Number+"|"+co.Language] = co
		}
	}
	return byNumber
}

// missingProductIDs maps a TCGplayer product id to the printing that product
// sells, for products whose printing the datastore holds without the id: The
// List products against their PLST row, and a few playtest cards. A product
// a card carries is priced from that card instead, so a row stops applying
// once the datastore carries its id.
var missingProductIDs = map[string]string{
	"485158": "0b81dfc8-5419-558b-a1d8-56001183c3b6", // Bringer of Green Zenith's Twilight, UNK RG02a
	"616942": "4b94ed4a-20c1-5e72-ba73-10fb1dd54fc1", // Second City, PF25 4
	"631720": "35739b26-c466-5548-b535-edd72a0c4cbc", // Chic // Ago, UNK CU12
	"637821": "ab8ec5f3-dd3c-5745-89c1-ac4a4a068ea5", // Arcane Teachings (Misprint), PLST JUD-78†
	"638130": "67762238-1162-5f97-91b4-518003ab329f", // Aerial Responder, PLST KLD-2
	"713783": "ac4c2971-b302-57d6-8c83-09e5edde2626", // Yeah Nah // Nah Yeah, UNK RU03e
	"721368": "ad2e1d3b-0add-501d-a214-0dc4945b3f7c", // Approach of the Second Sun, PLST Q06-1
	"721370": "0feb6411-d6f6-5254-8bde-c98a7445a4eb", // Marvo, Deep Operative, PLST MKC-7
	"721371": "2161f1d6-72b8-5e08-a00c-9aa5becb5bfd", // Claim Jumper, PLST OTJ-8
	"721372": "b4fe732c-d483-5ad8-a1e5-409877b772d2", // Leadership Vacuum, PLST C19-9
	"721374": "23a06d59-8b03-5a25-8ecb-a32a9469ad3f", // Swift Reconfiguration, PLST NEC-10
	"721385": "7e99246c-2ee6-5400-8d1c-8da08f5fc75f", // Vesuvan Drifter, PLST MAT-10
	"721386": "754f90a6-4476-51ba-8c9d-62bf61817bca", // Aerial Extortionist, PLST NCC-11
	"721387": "e5a96c36-c45b-53ae-b536-68c50eff14df", // Inniaz, the Gale Force, PLST JMP-12
	"721389": "837fc0e4-def4-59d1-8571-d16551e5bbf8", // Smuggler's Share, PLST NCC-21
	"721390": "936c9cc4-4489-5139-94f2-b3aebafbcb77", // Planeswalker's Mischief, PLST PLS-29
	"721392": "597e1da9-886d-52d1-bb71-7080b17dd57d", // Jeskai Infiltrator, PLST FRF-36
	"721393": "7acba392-82bc-533e-b4eb-c86c7e636778", // Fractured Identity, PLST C17-37
	"721394": "1f007c4f-1152-54b2-938e-a4f129295abc", // Exchange of Words, PLST UNF-45
	"721395": "52132750-00b4-5e6d-9c82-529078698e3e", // Aeon Engine, PLST C19-52
	"721397": "f8ac3874-98da-57af-a8cd-db627f28c2cc", // Master of Predicaments, PLST M15-67
	"721398": "189baf71-4725-5f26-bff1-c2eb3480484f", // Psychic Battle, PLST INV-68
	"721399": "b5f66d06-cef2-525b-b3d7-51b03acb1333", // Imprisoned in the Moon, PLST INR-69
	"721401": "ba18c632-d0d1-5ee1-8538-bea8f485a0f6", // Ponder, PLST DSC-73
	"721403": "36a5faf8-2556-5713-8be5-c17484103dd0", // Extinction Event, PLST IKO-88
	"721404": "e669979f-7741-5ac5-a22c-7c727998f49a", // Dazzling Sphinx, PLST OTC-93
	"721405": "9abb1048-c69d-5a15-93a7-bcca7347b1ce", // Rapid Hybridization, PLST CLU-93
	"721406": "e15678b9-5cd4-5fc2-974e-8f45db26f6a1", // Brainstorm, PLST MKC-96
	"721407": "8b1d8b3f-521d-5263-b783-b3f7f358072f", // Nadir Kraken, PLST MKC-112
	"721408": "af42f1c6-5648-5c83-a3b6-84a5f60f13e4", // Arcane Signet, PLST BLC-127
	"721409": "4e5cd30a-636a-5c63-ac0c-8262edfd45d2", // Doom Whisperer, PLST MKC-128
	"721415": "63b03fab-b421-50d3-a095-12e71e3064af", // Command Tower, PLST BLC-130
	"721416": "2f4545f8-45eb-5955-bc14-656d884e36c7", // Exotic Orchard, PLST BLC-131
	"721417": "f492c90c-8f1d-5033-963a-b63179a10b5d", // Toxrill, the Corrosive, PLST VOW-132
	"721418": "f9ef2af6-2368-519b-b205-31dcc2471362", // Loran of the Third Path, PLST BLC-143
	"721420": "5ef7d9fb-2aae-53f0-a3ab-a80f886d00ed", // Profane Transfusion, PLST CMR-145
	"721421": "658a123e-0321-54db-bd2a-9bf2c3ce346b", // Promise of Loyalty, PLST BLC-148
	"721422": "b29607f7-1842-5384-ba8b-3bb88e0feddf", // Skyclave Apparition, PLST BLC-154
	"721423": "8641c228-b43e-549f-a5cd-61b1fbf6b6a6", // Time Stop, PLST FDN-166
	"721424": "2a63637e-30ba-5f12-894a-eee1c59f2229", // Enigma Sphinx, PLST C18-178
	"721425": "c9fb0a7d-70a1-5fc8-8008-d46f92c5d77c", // Space Beleren, PLST UNF-178
	"721426": "49b16a55-2fef-511d-9ca2-934713b7c813", // Ethersworn Sphinx, PLST MH2-195
	"721427": "2ef2219b-f70f-554b-aa50-e6c9c3a3f9c9", // Breena, the Demagogue, PLST OTC-217
	"721428": "de6f56cd-7451-5d61-b171-b270c224c387", // Contaminated Landscape, PLST MH3-218
	"721429": "c905980d-3531-57fa-8927-20ccdc947daf", // Chaos Wand, PLST M19-228
	"721430": "df026d7b-379a-558c-a1b8-d39ef37a8ace", // Adarkar Wastes, PLST DMU-243
	"721431": "18e08eb6-8aa7-5400-b598-8e960d1731e6", // Thief of Sanity, PLST OTC-243
	"721432": "ded393d6-ce2d-56e0-ba53-361dafcc7e8b", // Caves of Koilos, PLST DMU-244
	"721433": "2405b709-b470-5553-83fb-fcc8c4483bdb", // Commander's Sphere, PLST DSC-244
	"721434": "0c91c77c-1751-5d30-b7a0-9c3a93b60f40", // Silent Clearing, PLST MH1-246
	"721435": "fbf01fee-3eed-5189-8886-d81fc53b268e", // Fabled Passage, PLST BLB-252
	"721436": "7b714e38-185f-5717-b0fc-914895418d76", // Arcane Sanctum, PLST DSC-259
	"721450": "19ac52e9-2003-5c33-9be3-329cac241a40", // Underground River, PLST BRO-267
	"721451": "d6072363-012b-546e-bcf0-4e7a4d5b6e2d", // Venser's Journal, PLST C21-273
	"721452": "52e12993-194a-56e9-87ca-95bfb4086874", // Command Beacon, PLST OTC-279
	"721453": "906af1de-95b2-5cc6-ada7-aaf9a23a8d0c", // Prairie Stream, PLST MKC-281
	"721454": "5c742312-e49a-535a-854a-d75460d758e0", // Coalition Relic, PLST M3C-286
	"721455": "a1727b83-a910-5f10-ba9d-204219cffbf7", // Myriad Landscape, PLST DSC-289
	"721456": "9e98ff2f-7d61-5484-accd-d2ed1961e879", // Drowned Catacomb, PLST OTC-290
	"721457": "0158c3f8-167c-5841-b7d7-deacf2113109", // Reliquary Tower, PLST DSC-295
	"721458": "bfc2fcc3-7ff2-55c0-abd0-58a10c220f45", // Mirage Mirror, PLST M3C-300
	"721459": "99cf9dda-6a88-5882-9d92-d0ce18c99da5", // Isolated Chapel, PLST OTC-301
	"721460": "ccd3bea1-22e9-5647-9b60-969ce42decd5", // Glacial Fortress, PLST BLC-307
	"721461": "d19b454f-b134-5a79-a7ea-1e38249d1c6b", // Temple of Deceit, PLST DSC-307
	"721462": "c9a6334b-2a74-5566-bfe4-b738896306dc", // Temple of Enlightenment, PLST DSC-308
	"721463": "5ce72954-6260-53c8-9bbc-ef035261d9db", // Temple of Silence, PLST DSC-312
	"721464": "a40f571c-d613-51fb-accb-a2569ba6968e", // Talisman of Progress, PLST M3C-313
	"721465": "f8970a7c-264d-522c-ba9e-dd8fe05e4ea5", // Wayfarer's Bauble, PLST M3C-315
	"721466": "748a5092-b06e-59b9-9a03-564e04282c9a", // Path of Ancestry, PLST BLC-322
	"721477": "b3e393c8-308d-5069-a737-605e8c3d5503", // Sunken Hollow, PLST OTC-325
	"721478": "9cc9ea1e-c1fd-5bc2-8205-119eed966954", // Evolving Wilds, PLST LCC-328
	"721479": "515c7530-07a3-5fc3-9c37-fe1b97e965ba", // Terramorphic Expanse, PLST BLC-345
	"721480": "bbf0955e-60e5-59c3-ae42-32f1858c57d6", // Mystic Gate, PLST M3C-359
	"721481": "ed184b24-7150-5e31-ae94-8a5d46c0ee05", // Skyclave Relic, PLST MOC-380
	"721482": "c3dea965-e5b5-5d37-ba5f-4ff33b0b7ba2", // Yavimaya, Cradle of Growth, PLST M3C-409
	"721483": "9a649be3-f1e3-5da5-9d7c-bd08520116b1", // Aboleth Spawn, PLST CLB-662
}

// productPrintings maps each product id, and the subtype its price rows
// carry, to the printing the datastore says that product sells in that
// finish: "Normal" for nonfoil, "Foil" for foil, and for etched on an etched
// product. A product claimed by more than one card keeps the first, the same
// way requesting each id once always did.
func productPrintings(b *mtgmatcher.Backend, collisions map[string][]string) map[string]map[string]string {
	printings := map[string]map[string]string{}
	add := func(id, subtype, cardID string) {
		if id == "" || cardID == "" || collisions[id] != nil {
			return
		}
		if printings[id] == nil {
			printings[id] = map[string]string{}
		}
		if _, found := printings[id][subtype]; !found {
			printings[id][subtype] = cardID
		}
	}

	for _, code := range b.GetAllSets() {
		set, err := b.GetSet(code)
		if err != nil {
			continue
		}
		byNumber := cardsByNumber(b, set)
		for _, card := range set.Cards {
			co, found := b.UUIDs[card.UUID]
			if !found {
				continue
			}
			id := co.Identifiers["tcgplayerProductId"]
			etchedID := co.Identifiers["tcgplayerEtchedProductId"]
			if soldAsCopy(b, id, co.Language) || soldAsCopy(b, etchedID, co.Language) {
				continue
			}
			if etchedID == "" {
				etchedID = id
			}
			add(id, "Normal", co.FoilUUIDs[mtgmatcher.FinishNonfoil])
			add(id, "Foil", co.FoilUUIDs[mtgmatcher.FinishFoil])
			add(etchedID, "Foil", co.FoilUUIDs[mtgmatcher.FinishEtched])
		}

		for _, base := range byNumber {
			co := starTwin(byNumber, &base.Card)
			if co == nil {
				continue
			}
			id := base.Identifiers["tcgplayerProductId"]
			add(id, "Normal", co.FoilUUIDs[mtgmatcher.FinishNonfoil])
			add(id, "Foil", co.FoilUUIDs[mtgmatcher.FinishFoil])
		}
	}

	for id, uuid := range missingProductIDs {
		co, found := b.UUIDs[uuid]
		if !found || printings[id] != nil {
			continue
		}
		add(id, "Normal", co.FoilUUIDs[mtgmatcher.FinishNonfoil])
		add(id, "Foil", co.FoilUUIDs[mtgmatcher.FinishFoil])
	}

	// A two-sided token sheet's combined entity is in no set's card list
	// (see mtgmatcher/magic/tokenpairs.go), one object per finish.
	for uuid, co := range b.UUIDs {
		if co.Identifiers["derivedTokenPair"] != "true" {
			continue
		}
		subtype := "Normal"
		if co.Foil || co.Etched {
			subtype = "Foil"
		}
		add(co.Identifiers["tcgplayerProductId"], subtype, uuid)
	}
	return printings
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (tcg *Index) Load(ctx context.Context) error {
	collisions := crossSetProductIDs(tcg.backend)
	for id, sets := range collisions {
		tcg.printf("skipping id %s, claimed by more than one set: %v", id, sets)
	}
	tcg.printings = productPrintings(tcg.backend, collisions)

	ids := slices.Collect(maps.Keys(tcg.printings))

	consume := func(result responseChan) {
		// Relaxed because sometimes we get duplicates due to how the ids
		// get buffered, but there is really no harm
		err := tcg.inventory.AddRelaxed(result.cardID, &result.entry)
		if err != nil {
			tcg.printf("%s", err.Error())
		}
	}
	mtgban.WorkerPool(ctx, tcg.maxConcurrency, slices.Collect(slices.Chunk(ids, tcgplayer.MaxIDsInRequest)),
		func(ctx context.Context, ids []string, channel chan<- responseChan) error {
			return tcg.processEntry(ctx, channel, ids)
		},
		consume,
		tcg.printf,
	)
	err := ctx.Err()
	if err != nil {
		return err
	}

	tcg.inventoryDate = time.Now()

	return nil
}

// Inventory returns what Load collected. See mtgban.Seller.
func (tcg *Index) Inventory() mtgban.InventoryRecord {
	return tcg.inventory
}

// MarketNames names the sub-sellers this market splits into. See
// mtgban.Market.
func (tcg *Index) MarketNames() []string {
	return availableIndexNames
}

// InfoForScraper describes one of the sub-scrapers named above.
func (tcg *Index) InfoForScraper(name string) mtgban.ScraperInfo {
	info := tcg.Info()
	info.Name = name
	info.Shorthand = name2shorthand[name]
	return info
}

// Info describes this scraper. See mtgban.Scraper.
func (tcg *Index) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Index"
	info.Shorthand = "TCGIndex"
	info.InventoryTimestamp = &tcg.inventoryDate
	info.MetadataOnly = true
	info.NoQuantityInventory = true
	info.Family = "TCG"
	info.Game = tcg.backend.Game
	return
}
