package vegassingles

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// uniqueCopy is the marker vegas.singles ends a display name with when the
// listing is one particular card rather than the printing: the word Unique,
// with or without parentheses, and then that copy's own id.
//
// The id is what makes the marker safe to read. Magic has a set named
// "Unique and Miscellaneous Promos", so every product in it says the word;
// none of them ends it with a number.
var uniqueCopy = regexp.MustCompile(`(?i)\(?Unique\)?\s*\(?\d+\)?$`)

// preprocess turns a storefront product into the matcher's input, in the
// grammar its game's display names follow.
//
// A listing for one particular copy is refused before any of that. It is a
// single card - signed, graded, or otherwise picked out - and it carries a
// price of its own, while the id it would resolve to is the printing's, held
// by the ordinary listing standing beside it. Publishing the copy's price
// under the printing's id lets a one-off set what the card is worth.
func preprocess(product VSProduct, game string) (*mtgmatcher.InputCard, error) {
	if uniqueCopy.MatchString(strings.TrimSpace(product.DisplayName)) {
		return nil, errors.New("listing is one particular copy, not the printing")
	}

	if displaySet(product.DisplayName) == oversizeHeading {
		return nil, errors.New("listing is an oversize display card, not a single")
	}

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

// cardTable spells the names the storefront types wrong. Each is a plain
// misspelling with nothing else to read it by - the card it means has
// printings and the name as written has none - so there is no rule to find
// here, only the correction.
var cardTable = map[string]string{
	"Stripe Mine":          "Strip Mine",
	"The Visioon":          "The Vision",
	"Thousand-Year Elixer": "Thousand-Year Elixir",
}

// magicCode is the "(SETCODE-NUMBER)" group a magic display name carries just
// ahead of the set name, like "(RVR-280)" or "(SLD-1800)". It is the last such
// group in the name: a Secret Lair spells its number a second time, bare and
// zero-padded, before the drop's own qualifiers.
//
// The List states the set size behind the number, "(LIST-234/249)", and the
// size is not part of it. Reading nothing at all there costs the listing more
// than the number: with no number the prose reading of the edition aliases,
// so the storefront's own code stands as the edition, and a code naming no
// set leaves the matcher with the whole history of the name to choose from.
var magicCode = regexp.MustCompile(`\(([A-Z0-9]+)-(\d+[a-zA-Z]?\x{2605}?)(?:/\d+)?\)`)

// saysMore reports whether the number a display name spells carries what the
// storefront's own field cannot. The field is an int, so a star and a letter
// both fall off it - WAR-184* and UST-147C arrive as 184 and 147, beside the
// plain printings they belong next to.
//
// Padding is not more. The field states the number without it and the display
// name with it, MH1-007 against a field saying 7, and where that is the whole
// difference the tidier spelling stands.
func saysMore(spelled, stated string) bool {
	return strings.TrimLeft(spelled, "0") != stated
}

// resolved returns the printing a card names, and nil when it names none. It
// matches a copy, so the matcher's own edits to the input stay in the probe.
func resolved(card mtgmatcher.InputCard) *mtgmatcher.CardObject {
	id, err := mtgmatcher.Match(&card)
	if err != nil {
		return nil
	}
	co, err := mtgmatcher.GetUUID(id)
	if err != nil {
		return nil
	}
	return co
}

func preprocessMagic(product VSProduct) (*mtgmatcher.InputCard, error) {
	// Display name format: "Hallowed Fountain (RVR-280) - Ravnica Remastered",
	// and with the printing's own wording standing between the two:
	// "Acererak the Archlich (Rainbow Foil) (SLD-1784) - Secret Lair Drop".
	//
	// SplitVariants is what every other magic scraper reads that shape with,
	// and it knows the names carrying a parenthesis of their own - "Dwight,
	// Assistant (to the) King", the B.F.M. halves, the tokens - which cutting
	// at the first bracket destroys.
	head := product.DisplayName
	if idx := strings.LastIndex(head, " - "); idx != -1 {
		head = head[:idx]
	}
	fields := mtgmatcher.SplitVariants(head)
	cardName := fields[0]
	if fixed, found := cardTable[cardName]; found {
		cardName = fixed
	}

	// What is left is the wording, less the code group: that is the number's
	// own business and is read below.
	var qualifiers []string
	for _, field := range fields[1:] {
		if magicCode.MatchString("(" + field + ")") {
			continue
		}
		// A wording naming a foil says nothing the tail has not already
		// said, and says it wrongly where the tail stayed silent: the
		// nonfoil "Endless Sands (0060) (Borderless) (Galaxy Foil)" is
		// sold beside its foil, and reading the treatment as a finish
		// puts both on the foil printing.
		if strings.Contains(strings.ToLower(field), "foil") {
			continue
		}
		qualifiers = append(qualifiers, field)
	}

	edition := string(product.ProductData.Set)
	if edition == "" {
		edition = product.ProductData.SetName
	}

	variant := ""
	if product.ProductData.CollectorNumberNormalized > 0 {
		variant = strconv.Itoa(product.ProductData.CollectorNumberNormalized)
	}

	// The finish is the last word of the display name, and the storefront's
	// own selectedFinish field disagrees with it for 1.6% of the catalog, in
	// both directions: every Secret Lair Drop (Borderless) foil says nonfoil,
	// and a handful of nonfoil listings say foil. Both spellings of one
	// printing then land on one uuid, where the foil price is the one the
	// sort keeps and the nonfoil printing is quoted at all.
	foil := strings.HasSuffix(product.DisplayName, " Foil")

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
	groups := magicCode.FindAllStringSubmatch(product.DisplayName, -1)
	if len(groups) > 0 {
		named := card
		named.Variation = groups[len(groups)-1][2]
		if saysMore(named.Variation, card.Variation) && resolved(named) != nil {
			card.Variation = named.Variation
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
	// printing at 268p while the storefront states the plain 268, so that
	// suffix is offered behind the bare number.
	//
	// The prose is a heading before it is a set, though, and a heading files
	// promos of every set together. Where it names one the listing's own card
	// does not belong to, the matcher still has to answer, so it leaves the
	// heading behind and reaches whatever printing it can - and that printing
	// is not the one the listing numbers. So the prose is trusted first for
	// the number it answers to, and only then for itself.
	//
	// Between the two stands the set the display name spells out behind its
	// last dash, which is the storefront's own prose for this listing rather
	// than for its whole bin: Snapcaster Mage at PTP-002 is filed under "Pro
	// Tour Promos" and named "Regional Championship Qualifiers 2023", where
	// the 2 the listing states stands.
	named := product.ProductData.SetName
	if named != "" && named != edition && !namesSet(edition, named) {
		prose, proseCo := readsAs(card, named)
		spelled, spelledCo := readsAs(card, spelledSet(product.DisplayName))
		switch {
		case answersNumber(proseCo, card.Variation):
			card = prose
		case spelledCo != nil:
			card = spelled
		// A prerelease printing is a product of its own, filed under the
		// storefront's own "Prerelease Cards" heading, and every listing that
		// is one states the number it is filed at. So a heading reaching one
		// at a number the listing never says is describing another card:
		// Spectacular Spider-Man at MEDIA-002 is the Marvel Legends insert
		// numbered 2, not the datestamped promo at 14s.
		case proseCo != nil && (card.Variation == "" || !proseCo.HasPromoType(magic.PromoTypePrerelease)):
			card = prose
		}
	}

	// The wording the name carried is the printing's own - the treatment a
	// Secret Lair drop is sold in, the frame a promo wears, the set a The
	// List reprint came from. Two listings of one card differ by nothing
	// else: "Acererak the Archlich (Rainbow Foil) (SLD-1784)" and "Acererak
	// the Archlich (SLD-1784)" state the same number, and dropping the
	// wording leaves them both asking for it.
	//
	// It is asked only where nothing else answered. A listing whose number
	// already reached a printing is not improved by being told more: the
	// extended-art Iroh is a media insert the number finds on its own, and
	// adding the words "Extended Art" walks it onto the showcase printing
	// of the same card instead.
	if len(qualifiers) > 0 && resolved(card) == nil {
		named := card
		named.Variation = strings.TrimSpace(card.Variation + " " + strings.Join(qualifiers, " "))
		if resolved(named) != nil {
			card = named
		}
	}

	// A display name says etched two ways: as its own tail, "Modern Horizons 2
	// Etched Foil", or as a qualifier standing with the card's other wording,
	// "Panharmonicon (Foil Etched) (2X2-562) - Double Masters 2022 Foil".
	// Neither reaches the matcher today - the finish field never says etched,
	// and the name is cut at its first parenthesis - so an etched listing
	// lands on its own plain foil printing, which is a different card at a
	// different price.
	//
	// Etched rides in the variation. Asking for it where the printing has no
	// etched sibling is worse than not asking: Backend.output clears both
	// finish flags and answers with the nonfoil printing, a third wrong card
	// rather than the second. So the reading is kept only when what answers
	// to it is etched.
	if strings.HasSuffix(product.DisplayName, " Etched Foil") || strings.Contains(product.DisplayName, "(Foil Etched)") {
		etched := card
		etched.Variation = strings.TrimSpace(card.Variation + " Etched")
		if co := resolved(etched); co != nil && co.Etched {
			card = etched
		}
	}

	return &card, nil
}

// oversizeHeading is what a display name spells for the storefront's oversize
// bin. The cards filed under it are the tournament-prize and display pieces,
// printed at four times the size and sold as curios rather than singles, and
// the matcher refuses them outright when it is told what they are.
//
// It is rarely told. The listing states the bin's code, "over", which names
// no set, and the matcher takes no set from a code it cannot read - so it
// answers with whichever ordinary printing the name and number reach, and an
// oversize Ambition's Cost is published at the price of the 8th Edition card
// anyone can play. Reading the heading here is what stops that.
//
// The bin holds real singles too: the storefront files the Planechase planes
// under it, and their display names say so. Only what spells the heading
// itself is refused.
const oversizeHeading = "Oversize Cards"

// displaySet is the set a display name spells out behind its last dash, as
// the storefront wrote it. spelledSet answers only for a name the datastore
// knows, which the storefront's own headings are not.
func displaySet(displayName string) string {
	idx := strings.LastIndex(displayName, " - ")
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(strings.TrimSuffix(displayName[idx+len(" - "):], " Foil"))
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

// readsAs returns the card as an edition string reads it, and the printing
// that reading lands on, or nil when it lands on none - an edition naming no
// set included, which would otherwise be read as no edition at all.
func readsAs(card mtgmatcher.InputCard, edition string) (mtgmatcher.InputCard, *mtgmatcher.CardObject) {
	if edition == "" {
		return card, nil
	}
	suffixes := []string{""}
	if card.Variation != "" {
		suffixes = append(suffixes, "p")
	}
	for _, suffix := range suffixes {
		probe := card
		probe.Edition = edition
		probe.Variation = card.Variation + suffix
		co := resolved(probe)
		if co != nil {
			return probe, co
		}
	}
	return card, nil
}

// answersNumber reports whether a printing is the one a listing's number
// names. A set spells a promo printing's number with a letter behind it, 268p
// for the promo pack and 14s for the prerelease, and the storefront states
// the bare number zero-padded or not, so the digits are what is compared. A
// listing stating no number is answered by no printing.
func answersNumber(co *mtgmatcher.CardObject, number string) bool {
	if co == nil || number == "" {
		return false
	}
	digits := strings.TrimRightFunc(co.Number, func(r rune) bool {
		return r < '0' || r > '9'
	})
	return strings.TrimLeft(digits, "0") == strings.TrimLeft(number, "0")
}

// setQualifier is a parenthetical riding behind the set a display name spells
// out, like the "(Borderless)" of "Regional Championship Qualifiers 2023
// (Borderless)" or the "(enchantment)" of "Duskmourn: House of Horror Promos:
// (enchantment)".
var setQualifier = regexp.MustCompile(`\s*\([^()]*\)$`)

// spelledSet returns the set a display name spells out behind its last dash,
// and "" when what stands there names none. Only a name a set answers to
// exactly is taken: GetSetByName settles for the nearest set it can reach,
// and the storefront's own headings ("Media Promos", "MagicFest Cards") all
// reach one that way.
func spelledSet(displayName string) string {
	idx := strings.LastIndex(displayName, " - ")
	if idx == -1 {
		return ""
	}
	tail := strings.TrimSuffix(displayName[idx+len(" - "):], " Foil")
	tail = strings.TrimSuffix(tail, " Etched")
	for {
		trimmed := setQualifier.ReplaceAllString(tail, "")
		if trimmed == tail {
			break
		}
		tail = trimmed
	}
	tail = strings.TrimRight(strings.TrimSpace(tail), ":,-")
	set, err := mtgmatcher.GetSetByName(tail)
	if err != nil || !mtgmatcher.Equals(set.Name, tail) {
		return ""
	}
	return tail
}

// riftboundNumber is the collector group riftbound display names carry, like
// "(301*/298)": the printing's own number, starred for the showcase variants,
// over the set size the matcher does not need.
//
// Two shapes lead with letters: the runes, numbered R1 through R6 with a
// letter behind them for each reprint and no set size at all, and the
// Vendetta signature promos, SP1 through SP6 over a set size of six. Both
// are printings the datastore holds, so refusing to read the number is the
// only thing keeping them off the shelf - and the storefront zero-pads
// where the datastore does not, which the matcher already reads through.
//
// Only a letter-led code may leave the set size out, which is why the two
// shapes are spelled as separate alternatives rather than one with the size
// made optional. A group of bare digits standing alone is not a number: the
// unique listings end their name with the copy's own id in parentheses, and
// reading that as a collector number would answer a card nobody listed.
var riftboundNumber = regexp.MustCompile(`\(([A-Z]+\d+[a-z]?\*?)(?:/\d+)?\)|\((\d+[a-z]?\*?)/\d+\)`)

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

	// Whichever alternative matched left the other's group unset
	start, end := loc[2], loc[3]
	if start < 0 {
		start, end = loc[4], loc[5]
	}

	head := product.DisplayName[:loc[0]]
	cardName := head
	idx := strings.Index(cardName, " (")
	if idx != -1 {
		cardName = cardName[:idx]
	}

	variation := product.DisplayName[start:end]
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
// parentheses. The denominator is lettered in the subsets a set prints
// alongside itself - the Trainer Gallery numbers TG11/TG30, the Shiny Vault
// SV81/SV94 - which is the shape the matcher's own numberTailRe accepts.
//
// Reading a narrower denominator does not fail where the number stands: the
// search is unanchored, so it walks past a number it cannot read and matches
// the era code sitting in the edition half instead, and the card is built
// from there - "Altaria TG11/TG30  - Holofoil" for a name and "SWSH12" for a
// number, which the matcher answers as an unknown card name.
var pokemonNumber = regexp.MustCompile(` ([A-Z]{0,5}\d+[a-zA-Z]?(?:/[A-Z]{0,4}\d+)?) `)

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
