package vegassingles

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// preprocess turns a storefront product into the matcher's input, in the
// grammar its game's display names follow.
func preprocess(product VSProduct, game string) (*mtgmatcher.InputCard, error) {
	switch game {
	case GameRiftbound:
		return preprocessRiftbound(product)
	case GameOnePiece:
		return preprocessOnePiece(product)
	case GamePokemon:
		return preprocessPokemon(product)
	}
	return preprocessMagic(product)
}

// magicCode is the "(SETCODE-NUMBER)" group a magic display name carries just
// ahead of the set name, like "(RVR-280)" or "(SLD-1800)". It is the last such
// group in the name: a Secret Lair spells its number a second time, bare and
// zero-padded, before the drop's own qualifiers.
var magicCode = regexp.MustCompile(`\(([A-Z0-9]+)-(\d+[a-zA-Z]?)\)`)

// resolves reports whether a card names a printing the datastore holds. It
// matches a copy, so the matcher's own edits to the input stay in the probe.
func resolves(card mtgmatcher.InputCard) bool {
	_, err := mtgmatcher.Match(&card)
	return err == nil
}

func preprocessMagic(product VSProduct) (*mtgmatcher.InputCard, error) {
	// Display name format: "Hallowed Fountain (RVR-280) - Ravnica Remastered"
	// Extract card name by finding the first " ("
	cardName := product.DisplayName
	if idx := strings.Index(cardName, " ("); idx != -1 {
		cardName = cardName[:idx]
	}

	edition := string(product.ProductData.Set)
	if edition == "" {
		edition = product.ProductData.SetName
	}

	variant := ""
	if product.ProductData.CollectorNumberNormalized > 0 {
		variant = strconv.Itoa(product.ProductData.CollectorNumberNormalized)
	}

	foil := product.SelectedFinish == "foil"

	card := mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
		Foil:      foil,
	}

	// The collector number is stated twice, and the int field is null for
	// whole sets: every Secret Lair drop, Edge of Eternities and its Stellar
	// Sights, the Countdown Kit. The display name's own code group always
	// spells it, so read it there when the field says nothing - a Secret Lair
	// with no number resolves to whichever drop the name aliases onto.
	//
	// Only what the field left empty is read this way, and only when it names
	// a printing: some codes are the storefront's own filing rather than a
	// set's, and their number counts within it ("(UMP-002) - Unique and
	// Miscellaneous Promos" is the second card the storefront filed there,
	// not a collector number), which no set answers to.
	if variant == "" {
		groups := magicCode.FindAllStringSubmatch(product.DisplayName, -1)
		if len(groups) > 0 {
			named := card
			named.Variation = groups[len(groups)-1][2]
			if resolves(named) {
				card.Variation = named.Variation
			}
		}
	}

	// The code is the storefront's filing and setName its prose, and for a
	// promo printing the two name different sets: a promo pack card is filed
	// under "ppotj" and named "Outlaws of Thunder Junction Promos", a
	// prerelease under "pre", a Timeshifts card under "mh1". The code names
	// the parent, so reading it writes the promo's buy price onto the
	// main-set card - Archangel of Tithes at PPOTJ-002 lands on OTJ 2, beside
	// the real OTJ 2 listing.
	//
	// So when the code names a set the prose does not, ask the prose. Only a
	// reading a printing answers to is taken: several names are the
	// storefront's own headings rather than a set's ("Oversize Cards",
	// "Marvel Eternal-Legal", "Unique and Miscellaneous Promos"), and the
	// code's reading is what those have.
	//
	// The number has to be said the promo way too. A set files its promo pack
	// printing at 268p and its prerelease at 268s while the storefront states
	// the plain 268, so the two suffixes are offered behind the bare number.
	named := product.ProductData.SetName
	if named != "" && named != edition && !namesSet(edition, named) {
		suffixes := []string{""}
		if card.Variation != "" {
			suffixes = append(suffixes, "p")
		}
		for _, suffix := range suffixes {
			promo := card
			promo.Edition = named
			promo.Variation = card.Variation + suffix
			if resolves(promo) {
				card = promo
				break
			}
		}
	}

	return &card, nil
}

// namesSet reports whether a set code names the set an edition string spells
// out. A code the datastore does not use names no set at all.
func namesSet(code, edition string) bool {
	set, err := mtgmatcher.GetSet(code)
	if err != nil {
		return false
	}
	return strings.EqualFold(set.Name, edition)
}

// riftboundNumber is the collector group riftbound display names carry, like
// "(301*/298)": the printing's own number, starred for the showcase variants,
// over the set size the matcher does not need.
var riftboundNumber = regexp.MustCompile(`\((\d+[a-z]?\*?)/\d+\)`)

// riftboundTag is a parenthetical qualifier a riftbound display name carries
// ahead of its collector number, like "(Signature)" or "(Champion)".
var riftboundTag = regexp.MustCompile(`\(([^)]+)\)`)

// A riftbound display name reads
//
//	Jinx - Loose Cannon (Signature) (301*/298) - Origins Foil
//
// The name stops at the first parenthesis, and the tags standing between it
// and the number ride behind the number in the variation. A tag often only
// restates what the starred number already says - (Signature) always travels
// with a star - but the promotional printings share one plain number and the
// tag is the whole difference: the organized play set files Rengar, Trophy
// Hunter at 120 twice, once as the champion card and once not.
func preprocessRiftbound(product VSProduct) (*mtgmatcher.InputCard, error) {
	loc := riftboundNumber.FindStringSubmatchIndex(product.DisplayName)
	if loc == nil {
		return nil, errors.New("no collector number in display name")
	}

	head := product.DisplayName[:loc[0]]
	cardName := head
	idx := strings.Index(cardName, " (")
	if idx != -1 {
		cardName = cardName[:idx]
	}

	variation := product.DisplayName[loc[2]:loc[3]]
	for _, tag := range riftboundTag.FindAllStringSubmatch(head, -1) {
		variation += " " + tag[1]
	}

	return &mtgmatcher.InputCard{
		Name:      strings.TrimSpace(cardName),
		Edition:   product.ProductData.SetName,
		Variation: strings.TrimSpace(variation),
		Foil:      strings.EqualFold(product.SelectedFinish, "foil"),
	}, nil
}

// onePieceCode is the card code One Piece display names carry, like
// "(OP06-020)" or "(P-037)", which names the printing on its own.
var onePieceCode = regexp.MustCompile(`\(([A-Z]+\d*-\d+[a-z]?)\)`)

// A One Piece display name reads
//
//	Hody Jones (020) (Alternate Art) (OP06-020) - Wings of the Captain Foil
//
// Everything before the code stays in the name: the matcher reads variant
// wording like (Alternate Art) or (Parallel) from it to pick the printing
// the storefront is describing.
func preprocessOnePiece(product VSProduct) (*mtgmatcher.InputCard, error) {
	loc := onePieceCode.FindStringSubmatchIndex(product.DisplayName)
	if loc == nil {
		return nil, errors.New("no card code in display name")
	}

	return &mtgmatcher.InputCard{
		Name:      strings.TrimSpace(product.DisplayName[:loc[0]]),
		Edition:   product.ProductData.SetName,
		Variation: product.DisplayName[loc[2]:loc[3]],
		Foil:      strings.EqualFold(product.SelectedFinish, "foil"),
	}, nil
}

// pokemonNumber is the collector number Pokemon display names carry inline,
// like "039/73", "042" or "SWSH197", set apart by spaces rather than
// parentheses.
var pokemonNumber = regexp.MustCompile(` ([A-Z]{0,5}\d+[a-zA-Z]?(?:/\d+)?) `)

// A Pokemon display name reads
//
//	Mewtwo GX 039/73  - Holofoil Shining Legends - Ultra Rare
//
// The name is what stands before the number, promo parentheticals like
// (SDCC 2007) included; the finish travels in its own field for this game,
// where the matcher tells Holofoil from Reverse Holofoil by wording.
func preprocessPokemon(product VSProduct) (*mtgmatcher.InputCard, error) {
	loc := pokemonNumber.FindStringSubmatchIndex(product.DisplayName)
	if loc == nil {
		return nil, errors.New("no collector number in display name")
	}

	finish := product.SelectedFinish
	if strings.EqualFold(finish, "Normal") {
		finish = ""
	}

	return &mtgmatcher.InputCard{
		Name:      strings.TrimSpace(product.DisplayName[:loc[0]]),
		Edition:   product.ProductData.SetName,
		Variation: product.DisplayName[loc[2]:loc[3]],
		Finish:    finish,
	}, nil
}
