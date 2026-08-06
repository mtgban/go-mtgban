package mtgmatcher

import "unique"

// The JSON decoder allocates a fresh string for every literal it meets,
// so the same set codes, artist names, finishes, legalities, and shared
// oracle text between reprints exist in thousands of copies right after
// decoding. The walkers below rewrite every string in the payload to
// its canonical copy from the runtime's interning table (the unique
// package), leaving one live copy of each distinct string. The table
// holds its entries weakly and the handles are dropped on the spot, so
// it empties itself at the next collection instead of living on as a
// cache.

func intern(s string) string {
	return unique.Make(s).Value()
}

func internSlice(ss []string) {
	for i := range ss {
		ss[i] = intern(ss[i])
	}
}

// internKeys re-inserts every entry of m under the canonical copy of
// its key. Assigning to an existing key does not replace the stored
// key, so each entry has to be deleted and added back; the keys are
// snapshotted first to keep the iteration away from the mutations.
func internKeys[V any](m map[string]V) {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	for _, key := range keys {
		val := m[key]
		delete(m, key)
		m[intern(key)] = val
	}
}

func internMap(m map[string]string) {
	for key, val := range m {
		m[key] = intern(val)
	}
	internKeys(m)
}

func internCard(card *Card) {
	card.Artist = intern(card.Artist)
	card.BorderColor = intern(card.BorderColor)
	internSlice(card.Colors)
	internSlice(card.ColorIdentity)
	card.FaceName = intern(card.FaceName)
	card.FaceFlavorName = intern(card.FaceFlavorName)
	card.FacePrintedName = intern(card.FacePrintedName)
	internSlice(card.Finishes)
	card.FlavorName = intern(card.FlavorName)
	card.FlavorText = intern(card.FlavorText)
	internSlice(card.FrameEffects)
	card.FrameVersion = intern(card.FrameVersion)
	internMap(card.Identifiers)
	card.Language = intern(card.Language)
	card.Layout = intern(card.Layout)
	card.Name = intern(card.Name)
	card.Number = intern(card.Number)
	card.OriginalReleaseDate = intern(card.OriginalReleaseDate)
	card.PrintedName = intern(card.PrintedName)
	card.PrintedType = intern(card.PrintedType)
	internSlice(card.Printings)
	internSlice(card.PromoTypes)
	card.Rarity = intern(card.Rarity)
	card.SetCode = intern(card.SetCode)
	for _, uuids := range card.SourceProducts {
		internSlice(uuids)
	}
	internKeys(card.SourceProducts)
	card.Side = intern(card.Side)
	internSlice(card.Subsets)
	internSlice(card.Types)
	internSlice(card.Subtypes)
	internSlice(card.Supertypes)
	card.UUID = intern(card.UUID)
	internMap(card.Legalities)
	internSlice(card.Variations)
	card.Watermark = intern(card.Watermark)

	for i := range card.ForeignData {
		foreign := &card.ForeignData[i]
		foreign.Name = intern(foreign.Name)
		foreign.Language = intern(foreign.Language)
		internMap(foreign.Identifiers)
		foreign.Type = intern(foreign.Type)
	}
}

func internBooster(booster *Booster) {
	for i := range booster.Boosters {
		internKeys(booster.Boosters[i].Contents)
	}
	// the map header is shared with the stored Sheet, so the keys can
	// be interned in place without writing the value back
	for _, sheet := range booster.Sheets {
		internKeys(sheet.Cards)
	}
	internKeys(booster.Sheets)
	booster.Name = intern(booster.Name)
}

func internSealedContents(contents []SealedContent) {
	for i := range contents {
		content := &contents[i]
		content.Code = intern(content.Code)
		content.Name = intern(content.Name)
		content.Set = intern(content.Set)
		content.UUID = intern(content.UUID)
		for _, config := range content.Configs {
			for _, extra := range config {
				internSealedContents(extra)
			}
			internKeys(config)
		}
	}
}

func internSealedProduct(product *SealedProduct) {
	product.Category = intern(product.Category)
	for _, contents := range product.Contents {
		internSealedContents(contents)
	}
	internKeys(product.Contents)
	internMap(product.Identifiers)
	product.Name = intern(product.Name)
	product.SetCode = intern(product.SetCode)
	product.ReleaseDate = intern(product.ReleaseDate)
	product.Subtype = intern(product.Subtype)
	product.UUID = intern(product.UUID)
}

func internDeckCards(cards []DeckCard) {
	for i := range cards {
		cards[i].UUID = intern(cards[i].UUID)
	}
}

func internSet(set *Set) {
	set.Code = intern(set.Code)
	for i := range set.Cards {
		internCard(&set.Cards[i])
	}
	set.KeyruneCode = intern(set.KeyruneCode)
	set.Name = intern(set.Name)
	set.ParentCode = intern(set.ParentCode)
	set.ReleaseDate = intern(set.ReleaseDate)
	set.TokenSetCode = intern(set.TokenSetCode)
	for i := range set.Tokens {
		internCard(&set.Tokens[i])
	}
	set.Type = intern(set.Type)

	for name, booster := range set.Booster {
		internBooster(&booster)
		set.Booster[name] = booster
	}
	internKeys(set.Booster)

	for i := range set.SealedProduct {
		internSealedProduct(&set.SealedProduct[i])
	}

	for i := range set.Decks {
		deck := &set.Decks[i]
		deck.Code = intern(deck.Code)
		internDeckCards(deck.Commander)
		internDeckCards(deck.MainBoard)
		internDeckCards(deck.DisplayCommander)
		internDeckCards(deck.Planes)
		internDeckCards(deck.Schemes)
		internDeckCards(deck.SideBoard)
		internDeckCards(deck.Tokens)
		deck.Name = intern(deck.Name)
		internSlice(deck.SealedProductUUIDs)
	}
}

// internAllPrintings dedupes every string in the freshly decoded payload.
func internAllPrintings(ap *AllPrintings) {
	for _, set := range ap.Data {
		internSet(set)
	}
}
