package cardtrader

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// namedID picks between the two ids a blueprint carries, preferring the
// scryfall one as before and only weighing the name where the vendor's own
// ids contradict each other.
//
// An id is the surest thing a storefront publishes, but the two are not
// checked against each other on the way in and a shelf can be filed a card
// off: Card Trader's eight Strixhaven promos each carry the scryfall id of
// the card before them, so "Exponential Growth" priced as Ecological
// Appreciation. It runs the other way too - two blueprints carry a
// TCGplayer id belonging to another card - so neither space can simply win.
//
// Where the ids name two different cards the blueprint's own name is the
// third thing said about it, and the id it agrees with is the one to keep.
// Where the name settles nothing, which is what a pair of tokens sold under
// one blueprint looks like, the order stands and nothing changes.
func namedID(scryfallID, tcgplayerID, cardName string) string {
	if scryfallID == "" {
		return tcgplayerID
	}
	if tcgplayerID == "" || scryfallID == tcgplayerID {
		return scryfallID
	}

	scryfallCard, err := mtgmatcher.GetUUID(scryfallID)
	if err != nil {
		return scryfallID
	}
	tcgplayerCard, err := mtgmatcher.GetUUID(tcgplayerID)
	if err != nil {
		return scryfallID
	}
	if mtgmatcher.Equals(scryfallCard.Name, tcgplayerCard.Name) {
		return scryfallID
	}

	namesScryfall := idNamesCard(scryfallCard, cardName)
	namesTCGplayer := idNamesCard(tcgplayerCard, cardName)
	if namesTCGplayer && !namesScryfall {
		return tcgplayerID
	}
	return scryfallID
}

// idNamesCard reports whether a card is the one a blueprint's wording names.
// A storefront writes the front of a two-faced card where the catalog spells
// both halves, so the front alone stands for the whole.
func idNamesCard(co *mtgmatcher.CardObject, cardName string) bool {
	if mtgmatcher.Equals(co.Name, cardName) || mtgmatcher.Equals(co.FaceName, cardName) {
		return true
	}
	front, _, split := strings.Cut(co.Name, " // ")
	return split && mtgmatcher.Equals(front, cardName)
}

// tokenPairNumberRes are the real shapes measured against Card Trader's
// own collector_number for a two-sided token blueprint, tried in order:
// a leading T/F/CT marker then both numbers ("T 05/20", "F 1/3",
// "CT 01/01"), both numbers then a shared total and marker ("024-027/031
// T"), and each face's own number-total-T marker separately ("05-014T /
// 03-014T"). None of these are collector numbers standing alone the way
// Card Kingdom's or Star City Games's skus embed one - Card Trader
// composites both faces' own numbers into the one field it publishes,
// which is exactly why a plain single-number anchor (the shape every
// other vendor's fallback already handles) never applied here.
var tokenPairNumberRes = []*regexp.Regexp{
	regexp.MustCompile(`^(?:T|F|CT)?\s*0*(\d+)\s*/\s*0*(\d+)$`),
	regexp.MustCompile(`^0*(\d+)-0*(\d+)/\d+\s*T$`),
	regexp.MustCompile(`^0*(\d+)-\d+T\s*/\s*0*(\d+)-\d+T$`),
}

// tokenPairNumbers parses a two-sided token blueprint's own composite
// collector_number into each face's own number. Returns ok=false for a
// shape none of tokenPairNumberRes recognizes rather than guess at one -
// refuse, don't parse a shape never measured against the real catalog.
func tokenPairNumbers(number string) (n1, n2 string, ok bool) {
	number = strings.TrimSpace(number)
	for _, re := range tokenPairNumberRes {
		if m := re.FindStringSubmatch(number); m != nil {
			return m[1], m[2], true
		}
	}
	return "", "", false
}

// TokenPairAnchorUUIDs anchors both faces of a two-sided token blueprint
// independently by identity, given the blueprint's own composite
// collector_number (tokenPairNumbers) and its own claimed edition - rather
// than the single-number-anchored-against-the-whole-name lookups Preprocess
// itself uses above, which only resolve a pairing already known one way or
// another (native or previously derived). Exists purely for offline
// candidate discovery (cmd/tokenpairgen): finding two-sided listings a real
// pairing exists for that neither MatchNativeTokenPair nor
// MatchTokenPairingBySetNumber can find yet, because no pairing is on file
// for it at all. Not part of Preprocess's own resolution chain - the
// already-shipped, already-measured number-anchored fallback there is
// left untouched.
//
// Which parsed number names which face is not fixed across the shapes
// tokenPairNumbers recognizes (see its own comment), so both pairings of
// (n1, n2) against (first, second) are tried; ok is false unless exactly
// one of the two resolves both faces to one printing each - the same
// "don't know, refuse" the rest of this package already applies, now
// applied to which assignment is right rather than just whether a
// printing exists.
func TokenPairAnchorUUIDs(bp Blueprint) (uuidA, uuidB string, ok bool) {
	if bp.CategoryID != CategoryMagicTokens || !strings.Contains(bp.Name, " // ") {
		return "", "", false
	}
	n1, n2, parsed := tokenPairNumbers(bp.Properties.Number)
	if !parsed {
		return "", "", false
	}
	tokenSet := magic.EditionTokenSetCode(bp.Expansion.Name)
	if tokenSet == "" {
		return "", "", false
	}
	first, second := magic.SplitTokenPairName(bp.Name)
	rawFaces := [2]string{first, second}

	anchorOne := func(numbers [2]string) (string, string, bool) {
		var uuids [2]string
		for i, raw := range rawFaces {
			var uuid string
			for _, face := range []string{magic.StripFaceWrapping(raw), magic.CleanFaceName(raw)} {
				if cards := mtgmatcher.MatchInSetNumber(face, tokenSet, numbers[i]); len(cards) == 1 {
					uuid = cards[0].UUID
					break
				}
			}
			if uuid == "" {
				return "", "", false
			}
			uuids[i] = uuid
		}
		return uuids[0], uuids[1], true
	}

	uA, uB, okFwd := anchorOne([2]string{n1, n2})
	uA2, uB2, okRev := anchorOne([2]string{n2, n1})
	switch {
	case okFwd && !okRev:
		return uA, uB, true
	case okRev && !okFwd:
		return uA2, uB2, true
	default:
		return "", "", false
	}
}

// Preprocess turns a blueprint into the card description the matcher takes,
// reporting an error for the blueprints that are not cards.
func Preprocess(bp *Blueprint) (*mtgmatcher.InputCard, error) {
	cardName := bp.Name
	edition := bp.Expansion.Name
	number := strings.TrimLeft(bp.Properties.Number, "0")
	variant := ""

	// A two-sided token sheet's product ("Beast // Plant") is exactly the
	// shape namedID below silently mis-resolves: whichever single face one
	// of the blueprint's own ids happens to name, rather than the combined
	// pairing the name actually describes - the same "silently prices the
	// whole two-sided product as if it were just the one face" risk
	// documented in cardkingdom's and starcitygames's own versions of this
	// check (see magic.MatchTokenPairing's doc comment, shared by all
	// three). Card Trader differs from both in one way worth knowing: its
	// own TCGplayerID is frequently the *pairing's own* product id
	// directly, not one face's - magic.VerifyTokenPairingFinish checks
	// that case explicitly rather than relying on namedID's fallthrough to
	// stumble onto it by accident, which it otherwise would for a
	// scryfallID-less blueprint (measured: true for most, not all).
	// Foil is always false here: a blueprint has no finish of its own,
	// multiple products of different finishes share one, so it is
	// resolved per-product in cardtrader.go instead (see foilPrintingID's
	// own derived-pairing handling).
	//
	// Gated to CategoryMagicTokens rather than every " // " blueprint:
	// checked against the live catalog (122,480 blueprints, 6,077 named
	// "X // Y") that this isn't narrower than it looks. Every blueprint
	// whose own identifiers/name actually resolve through
	// magic.MatchTokenPairing or magic.VerifyTokenPairingFinish -
	// dungeon-card pairings included ("Dungeon of the Mad Mage // Lost
	// Mine of Phandelver") - was filed under CategoryMagicTokens; nothing
	// in CategoryMagicSingles/Oversized/Sleeves/Albums/etc. resolved.
	// CategoryMagicSingles' own 2,148 " // " blueprints are ordinary
	// split/transform/DFC cards whose real name is "X // Y" ("Turn //
	// Burn"), already served correctly by namedID below; the merchandise
	// categories' hits quote a card's name as product flavor text, not
	// cards at all. Re-check this if Card Trader ever starts selling a
	// two-sided pairing shape outside Tokens.
	if bp.CategoryID == CategoryMagicTokens && strings.Contains(cardName, " // ") {
		if tcgID := magic.MatchTokenPairing(bp.ScryfallID, cardName, false); tcgID != "" {
			if id, err := mtgmatcher.MatchID(tcgID, false); err == nil {
				return &mtgmatcher.InputCard{
					ID: id, Name: cardName, Edition: edition, Variation: bp.Version,
				}, nil
			}
		}
		if verified := magic.VerifyTokenPairingFinish(fmt.Sprintf("%d", bp.TCGplayerID), false); verified != "" {
			if id, err := mtgmatcher.MatchID(verified, false); err == nil {
				return &mtgmatcher.InputCard{
					ID: id, Name: cardName, Edition: edition, Variation: bp.Version,
				}, nil
			}
		}

		// Neither id resolved. Card Trader's own collector_number for a
		// two-sided token blueprint is not one face's number - it
		// composites both (see tokenPairNumbers) - so where it parses,
		// each face can be anchored by identity the same way Card
		// Kingdom's and Star City Games's sku-embedded numbers already
		// are: try the real printing mtgjson already files under one
		// combined name (magic.MatchNativeTokenPair), then the sku-
		// anchored derived-pairing fallback (magic.MatchTokenPairingBySetNumber,
		// the same mechanism both other vendors' own number-anchored
		// paths use), trying each parsed number in turn since which face
		// the first number names is not fixed across the shapes measured.
		if n1, n2, ok := tokenPairNumbers(bp.Properties.Number); ok {
			if tokenSet := magic.EditionTokenSetCode(edition); tokenSet != "" {
				for _, number := range []string{n1, n2} {
					if uuid := magic.MatchNativeTokenPair(tokenSet, number, cardName); uuid != "" {
						if id, err := mtgmatcher.MatchID(uuid, false); err == nil {
							return &mtgmatcher.InputCard{
								ID: id, Name: cardName, Edition: edition, Variation: bp.Version,
							}, nil
						}
					}
					if tcgID := magic.MatchTokenPairingBySetNumber(tokenSet, number, cardName, false); tcgID != "" {
						if id, err := mtgmatcher.MatchID(tcgID, false); err == nil {
							return &mtgmatcher.InputCard{
								ID: id, Name: cardName, Edition: edition, Variation: bp.Version,
							}, nil
						}
					}
				}
			}
		}

		// Still nothing - both faces' own names alone, guarded by
		// requiring the blueprint's own claimed edition to independently
		// agree with the match - see magic.MatchTokenPairingByNamesAndEdition's
		// own doc comment for why the guard is not optional (measured:
		// 11.6% of name-only matches against Card Trader's real catalog
		// would otherwise be silently wrong). Tried last: a real number
		// anchor above is strictly the safer bar when one parses.
		if tcgID := magic.MatchTokenPairingByNamesAndEdition(cardName, edition, false); tcgID != "" {
			if id, err := mtgmatcher.MatchID(tcgID, false); err == nil {
				return &mtgmatcher.InputCard{
					ID: id, Name: cardName, Edition: edition, Variation: bp.Version,
				}, nil
			}
		}
	}

	// Some, but not all, have a proper id we can reuse right away, and the
	// blueprint says which space each one lives in
	scryfallID := mtgmatcher.ConvertID(mtgmatcher.IDSpaceScryfall, bp.ScryfallID)
	tcgplayerID := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, fmt.Sprintf("%d", bp.TCGplayerID))
	id := namedID(scryfallID, tcgplayerID, cardName)
	if id != "" {
		return &mtgmatcher.InputCard{
			ID: id,
			// Not needed, but helps debugging
			Name:    cardName,
			Edition: edition,
			// Needed to detect etched finish
			Variation: bp.Version,
		}, nil
	}

	switch edition {
	case "Alliances", "Fallen Empires", "Homelands",
		"Guilds of Ravnica",
		"Ravnica Allegiance",
		"Kaldheim",
		"Asia Pacific Land Program",
		"European Land Program",
		"Commander's Arsenal",
		"Commander Anthology Volume II",
		"Unglued",
		"Mystical Archive: Japanese alternate-art",
		"Chronicles",
		"Chronicles Japanese",
		"Rinascimento",
		"Antiquities":
		variant = number
	case "Commander Legends: Commander Decks":
		edition = "Commander Legends"
		variant = number
	case "Arabian Nights":
		if strings.HasSuffix(number, "b") {
			variant = "light"
		} else if strings.Contains(number, "a") {
			variant = "dark"
		}
	case "Champions of Kamigawa":
		if cardName == "Brothers Yamazaki" {
			if bp.Version == "1" {
				variant = "160a"
			} else if bp.Version == "2" {
				variant = "160b"
			}
		}
	case "Buy a Box ", "Buy a Box",
		"Armada Comics",
		"Prerelease Promos":
		variant = edition
		if cardName == "Voldaren Estate" {
			variant = "Dracula"
		}
	case "Factory Misprints":
		variant = edition
		switch cardName {
		case "Laquatus's Champion":
			variant = "prerelease misprint"
		case "Island":
			edition = "PAL99"
		}
	case "Judge Gift Cards",
		"Wizards Play Network",
		"Arena League Promos",
		"Friday Night Magic",
		"Player Rewards Promos":
		ed, found := id2edition[bp.ID]
		if found {
			edition = ed
		}
		if bp.ID == 29063 {
			variant = "1"
		} else if bp.ID == 29053 {
			variant = "11"
		}
	case "Champs and States":
		if cardName == "Crucible of Worlds" {
			edition = "World Championship Promos"
		}
	case "Core Set 2021":
		if cardName == "Teferi, Master of Time" {
			variant = number
		}
	case "DCI Promos":
		switch cardName {
		case "Cryptic Command":
			edition = "PPRO"
		case "Flooded Strand":
			edition = "PNAT"
		}
	case "Grand Prix Promos":
		if cardName == "Wilt-Leaf Cavaliers" {
			edition = "DCI"
		}
	case "The List":
		switch cardName {
		case "Everythingamajig", "Ineffable Blessing":
			variant = strings.Fields(bp.Version)[0]
		default:
			if len(mtgmatcher.MatchInSetNumber(cardName, "PLST", number)) > 0 {
				variant = number
			}
		}
	case "Mystery Booster: Convention Edition Playtest Cards":
		variant = bp.Version
	case "Modern Horizons 2",
		"Modern Horizons 1: Timeshifted":
		variant = number
		if strings.HasSuffix(variant, "e") {
			variant = strings.TrimSuffix(variant, "e")
			variant += " Etched"
		}
	case "Commander: The Lord of the Rings - Tales of Middle-earth Collectors",
		"The Lord of the Rings: Tales of Middle-earth Holiday Release":
		variant = strings.Replace(number, "s", "z", 1)
	case "Simplified Chinese Alternate Art Cards":
		switch cardName {
		case "Drudge Skeletons":
			if bp.Version != "" {
				edition = bp.Version
			}
			variant = number
			if !strings.HasSuffix(variant, "s") {
				variant += "s"
			}
		}
	default:
		if strings.HasPrefix(edition, "Secret Lair") {
			variant = number
		} else if strings.HasSuffix(edition, "Collectors") {
			variant = number
		} else if strings.HasPrefix(edition, "WCD") ||
			strings.HasPrefix(edition, "Pro Tour 1996") {
			variant = number
			if strings.HasPrefix(variant, "sr") {
				variant = strings.Replace(variant, "sr", "shr", 1)
			}

			switch bp.ID {
			case 25481: // Scrabbling Claws
				variant = "jn237sb"
			case 25491: // Chrome Mox
				variant = "mb152"
			case 25501: // Seething Song
				variant = "ap104sb"
			case 32184: // Aura of Silence
				variant = "bh7bsb"
			case 35075: // Shatter
				variant = "gb219sb"
			}
		} else if strings.Contains(edition, "Japanese") {
			variant = "Japanese"
			if strings.Contains(edition, "Promo") {
				variant += " Prerelease"
			}
		} else if strings.HasSuffix(edition, "Promos") {
			variant = number

			switch edition {
			case "Guilds of Ravnica Promos":
				switch cardName {
				case "Attendant of Vraska",
					"Kraul Raider",
					"Precision Bolt",
					"Ral's Dispersal",
					"Ral's Staticaster",
					"Ral, Caller of Storms",
					"Vraska's Stoneglare",
					"Vraska, Regal Gorgon":
					edition = "Guilds of Ravnica"
				}
			// Starting from RNA, the collector numbers do not have the
			// right suffix any more, we need to special case a lot of cards
			case "Ravnica Allegiance Promos":
				switch cardName {
				case "Light Up the Stage",
					"Growth Spiral",
					"Mortify",
					"Rakdos Firewheeler",
					"Simic Ascendancy":
					variant = number
				case "Dovin, Architect of Law",
					"Elite Arrester",
					"Dovin's Dismissal",
					"Dovin's Automaton",
					"Domri, City Smasher",
					"Ragefire",
					"Charging War Boar",
					"Domri's Nodorog":
					edition = "Ravnica Allegiance"
				default:
					variant = number + "s"
				}
			case "War of the Spark Promos":
				switch cardName {
				case "Augur of Bolas",
					"Liliana's Triumph",
					"Paradise Druid",
					"Dovin's Veto":
					variant = number
				case "Bolas's Citadel",
					"Karn's Bastion":
					variant = number
					if bp.ID == 56810 || bp.ID == 56746 {
						variant = "Prerelease"
					} else if bp.ID == 60193 || bp.ID == 105989 {
						variant = "Promo Pack"
					}
				case "Feather, the Redeemed":
					variant = "Promo Pack"
					if bp.ID == 56782 {
						variant = "Prerelease"
					}
				case "Desperate Lunge",
					"Gideon's Battle Cry",
					"Gideon's Company",
					"Gideon, the Oathsworn",
					"Guildpact Informant",
					"Jace's Projection",
					"Jace's Ruse",
					"Jace, Arcane Strategist",
					"Orzhov Guildgate",
					"Simic Guildgate",
					"Tezzeret, Master of the Bridge":
					edition = "War of the Spark"
				default:
					variant = "Prerelease"
				}
			// This set acts mostly as a catch-all for anything prior :(
			case "Core Set 2020 Promos":
				version := mtgmatcher.ExtractNumber(strings.Replace(bp.Slug, "-", " ", -1))
				if mtgmatcher.IsBasicLand(cardName) {
					edition = "M20 Promo Packs"
				} else if cardName == "Chandra's Regulator" {
					if version == "1" {
						variant = number
					} else if version == "2" {
						variant = "Promo Pack"
					} else if version == "3" {
						variant = "Prerelease"
					}
				} else if version == "1" {
					variant = "Promo Pack"
				} else if version == "2" {
					variant = "Prerelease"
				} else {
					if magic.HasPromoPackPrinting(cardName) {
						variant = "Promo Pack"
						if cardName == "Sorcerous Spyglass" {
							edition = "PXLN"
						}
					} else {
						edition = "Core Set 2020"
					}
				}
			case "D&D: Adventures in the Forgotten Realms Promos":
				variant = "Promo Pack"
				edition = "PAFR"
			default:
				set, err := mtgmatcher.GetSet(bp.Expansion.Code)
				if err != nil {
					return nil, err
				}
				setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
				if err != nil {
					return nil, err
				}

				if setDate.After(magic.PromosForEverybodyYay) {
					notPromoPack := false
					num, convErr := strconv.Atoi(number)
					parentSet, setErr := mtgmatcher.GetSet(set.ParentCode)
					if convErr == nil && setErr == nil {
						notPromoPack = num > parentSet.BaseSetSize
					}

					if magic.HasPromoPackPrinting(cardName) && !notPromoPack {
						variant = "Promo Pack"
					} else {
						edition = strings.TrimSuffix(edition, " Promos")
					}
				} else {
					switch edition {
					case "Hour of Devastation Promos":
						if cardName == "Nicol Bolas, God-Pharaoh" && number == "140" {
							variant = "Prerelase"
						}
					case "Dominaria Promos":
						if cardName == "Steel Leaf Champion" && bp.ID == 1833 {
							variant = "182"
						}
					}
				}
			}
		}
	}

	if mtgmatcher.IsBasicLand(cardName) {
		variant = number
		switch edition {
		case "International Edition",
			"Introductory Two-Player Set",
			"Collectors’ Edition":
			return nil, errors.New("pass")
		// Some basic land foil are mapped to the Promos
		case "Guilds of Ravnica Promos",
			"Ravnica Allegiance Promos":
			edition = strings.TrimSuffix(edition, " Promos")
			if strings.HasPrefix(variant, "A") {
				edition = "GRN Ravnica Weekend"
			} else if strings.HasPrefix(variant, "B") {
				edition = "RNA Ravnica Weekend"
			}
		// Some lands have years set
		case "Arena League Promos":
			variant = mtgmatcher.ExtractYear(strings.Replace(bp.Slug, "-", " ", -1))

			switch variant {
			case "2001":
				switch cardName {
				case "Forest":
					variant = "2001 1"
				case "Mountain", "Swamp":
					variant = "2000"
				}
			case "2002":
				switch cardName {
				case "Forest":
					variant = "2001 11"
				case "Mountain", "Swamp":
					variant = "2001"
				}
			}
		case "Magic Premiere Shop":
			if number == "" {
				number = fmt.Sprint(bp.ID)
			}
			variant = pmpsTable[number]
		}
	}

	if strings.Contains(edition, "Prerelease") {
		edition = strings.Replace(edition, "Prerelease", "Promos", 1)
		variant = "Prerelease"

		switch cardName {
		case "Lu Bu, Master-at-Arms":
			edition = "Prerelease Events"
			if number == "6" {
				variant = "April"
			} else if number == "8" {
				variant = "July"
			}
		case "Chord of Calling", "Wrath of God":
			edition = "Double Masters"
			variant = number
		case "Magic Missile":
			edition = "ARF"
			variant = "401"
		}
	} else if strings.HasSuffix(edition, "Theme Deck") {
		edition = strings.TrimSuffix(edition, " Theme Deck")
	}

	if strings.Contains(bp.Version, "Etched") && !strings.Contains(variant, "Etched") {
		if variant != "" {
			variant += " "
		}
		variant += "Etched"
	}

	// Serialized uses a different suffix than scryfall
	if strings.Contains(bp.Version, "Serialized") {
		variant = "Serialized"
	}

	// Make sure the token tag is always present
	if bp.CategoryID == CategoryMagicTokens && !strings.Contains(cardName, "Token") {
		cardName += " Token"
		if variant == "" {
			variant = strings.TrimPrefix(number, "T")
		}
	}
	if bp.CategoryID == CategoryMagicOversized {
		switch {
		// Preserve the Display Commander tag
		case strings.Contains(bp.Version, "Display"):
			variant += " Display"
		// Make sure the oversize tag is always present
		case !strings.Contains(edition, "Oversize"):
			edition += " Oversize"
		}
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
	}, nil
}
