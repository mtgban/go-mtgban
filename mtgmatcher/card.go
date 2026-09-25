package mtgmatcher

import (
	"fmt"
	"slices"
	"strings"
)

// InputCard is a card as a storefront described it: a name plus whatever
// edition and variation text they published, which the matcher resolves to one
// printing. Field names follow the MTGJSON project.
type InputCard struct {
	// The unique identifier of the card
	// When used as input it can host or scryfall id
	ID string `json:"id,omitempty"`

	// The canonical name of the card
	Name string `json:"name,omitempty"`

	// The hint or commonly know variation
	Variation string `json:"variant,omitempty"`

	// The set the card comes from, or a portion of it
	Edition string `json:"edition,omitempty"`

	// Whether the card is foil or not
	Foil bool `json:"foil,omitempty"`

	// The finish being priced, spelled the way the source spells it
	// ("Normal", "Cold Foil", "Holofoil", …) and read through FinishSlug, so
	// a caller pricing one printing in
	// one finish can name the finish instead of choosing between the two the
	// Foil flag has a bit for. It refines the flag rather than replacing it,
	// and saying nothing leaves the flag to answer alone.
	Finish string `json:"finish,omitempty"`

	// The card belongs to the extended side of the set, usually containing
	// variants with the same name of existing cards in the same set, but with
	// different frames or border effects
	// Internal matcher state, not part of the serialized input.
	BeyondBaseSet bool `json:"-"`

	// In case edition information is not accurate, use this flag to
	// perform a best-effort search, which will try to isolate promo
	// printings from the others
	PromoWildcard bool `json:"PromoWildcard,omitempty"`

	// In case card got renamed in some way, this contains the original
	// card name, instead of the sanitized version
	// Internal matcher state, not part of the serialized input.
	OriginalName string `json:"-"`

	// The language as parsed
	Language string `json:"language,omitempty"`
}

// Card implements the Stringer interface
func (c *InputCard) String() string {
	name := c.Name
	edition := c.Edition

	if c.Variation != "" {
		name = fmt.Sprintf("%s ('%s')", name, c.Variation)
	}
	finish := ""
	if c.IsEtched() {
		finish = " (etched)"
	} else if c.Foil {
		finish = " (foil)"
	}
	lang := ""
	if c.Language != "" && c.Language != "English" {
		lang = " {" + c.Language + "}"
	}
	return fmt.Sprintf("%s [%s%s]%s", name, edition, finish, lang)
}

// cardDescription is what a diagnostic prints in place of an input card. The
// lookup runs when the line is formatted rather than when it is written, so a
// Match with no Logger set pays nothing for a description nobody reads.
type cardDescription struct {
	backend *Backend
	card    *InputCard
}

func (d cardDescription) String() string {
	if d.card.Name != "" {
		return d.card.String()
	}
	// The same lookup Match runs, so an external id the scraper handed
	// over describes as the printing it names, not as an unknown uuid.
	co, err := d.backend.cardObject4Id(d.card.ID)
	if err != nil {
		return d.card.String()
	}
	named := *d.card
	named.Name = co.Name
	named.Edition = co.Edition
	return named.String()
}

// describe names the card a diagnostic is about. An input that carries only
// an id has no name for String to print, and String has no datastore to
// resolve one from; this is the same description made where there is one.
func (b *Backend) describe(c *InputCard) fmt.Stringer {
	return cardDescription{b, c}
}

// AddToVariant appends a tag to the variation, keeping what is already there
// and separating with a space.
func (c *InputCard) AddToVariant(tag string) {
	if c.Variation != "" {
		c.Variation += " "
	}
	c.Variation += tag
}

// The Is* predicates below read the free text a storefront published, not the
// datastore: they decide what a listing claims, and the rules then decide
// whether a printing can honour the claim.
//
// Two spellings appear throughout and they differ in reach. c.Contains(x)
// looks in both Edition and Variation, while Contains(c.Variation, x) looks
// only at the variation, which matters when an edition name would otherwise
// match every card in it. Both fold case and punctuation, and a few
// predicates need the raw strings.Contains instead, and say so where they do.
//
// The vendor abbreviations in the clauses name the storefront whose wording
// forced that clause: each is a real listing someone published.

// IsFoil reports a foil printing from the variation, refusing Non-Foil and
// leaving etched to IsEtched.
func (c *InputCard) IsFoil() bool {
	return Contains(c.Variation, "Foil") && !Contains(c.Variation, "Non") && !c.IsEtched()
}

// IsEtched reports etched foiling. It matches the whole word because the stem
// would also catch Sketch.
func (c *InputCard) IsEtched() bool {
	// Note this can't be just "etch" because it would catch the "sketch" cards
	return Contains(c.Variation, "Etched")
}

// Contains reports whether either the edition or the variation contains the
// property, ignoring case and punctuation. Prefer Contains(c.Variation, prop)
// where an edition name would match every card printed in it.
func (c *InputCard) Contains(prop string) bool {
	return Contains(c.Edition, prop) || Contains(c.Variation, prop)
}

// Equals reports whether either the edition or the variation is exactly the
// property, ignoring case and punctuation.
func (c *InputCard) Equals(prop string) bool {
	return Equals(c.Edition, prop) || Equals(c.Variation, prop)
}

// IsToken reports whether the name is a token in this datastore. The names a
// game knows as tokens without carrying a token type of their own are its own
// business, so the game's rules are asked as well.
func (b *Backend) IsToken(name string) bool {
	if slices.Contains(b.Tokens, name) {
		return true
	}
	if b.rules == nil {
		return false
	}
	return b.rules.IsToken(b, name)
}

// PlainNumber reduces a collector number the way this datastore's game does.
// A datastore with no rules attached hands the number back.
func (b *Backend) PlainNumber(number string) string {
	if b.rules == nil {
		return number
	}
	return b.rules.PlainNumber(number)
}

func (b *Backend) output(card Card, flags ...bool) string {
	hasNonfoil := card.HasFinish(FinishNonfoil)
	hasFoil := card.HasFinish(FinishFoil)
	hasEtched := card.HasFinish(FinishEtched)

	// The etched flag is answered only by a card sold etched. One that is
	// not keeps the foil flag beside it: that is the finish the caller
	// asked for next, not none at all.
	etched := len(flags) > 1 && flags[1] && hasEtched
	foil := len(flags) > 0 && flags[0] && !etched

	// In case the foiling information is incorrect
	if !foil && !hasNonfoil && !hasEtched {
		foil = true
	} else if foil && !hasFoil {
		foil = false
	}
	if hasFoil && !hasNonfoil && !hasEtched {
		foil = true
	} else if !hasFoil && (hasNonfoil || hasEtched) {
		foil = false
	}

	// In case the etching information is incorrect
	if !etched && !hasNonfoil && !hasFoil {
		etched = true
	}
	if hasEtched && !hasNonfoil && !hasFoil {
		etched = true
	} else if !hasEtched && (hasNonfoil || hasFoil) {
		etched = false
	}

	// Resolve to the finish the caller is asking for and pull the uuid the
	// loader registered for it. Loaders register every finish a card carries
	// (and, for Lorcana, every foil sub-type), so this is the common path.
	finish := FinishNonfoil
	if etched {
		finish = FinishEtched
	} else if foil {
		finish = FinishFoil
	}
	if id, ok := card.FoilUUIDs[finish]; ok {
		return id
	}

	// Fall back to the suffix rules for cards without a registered map.
	id := card.UUID
	// Append suffixes to the Id to distinguish cards among finishes
	if etched && (hasNonfoil || hasFoil) {
		// Retrieve the base id if it's already tagged (only for this and the case below)
		if strings.HasSuffix(id, suffixFoil) || strings.HasSuffix(id, suffixEtched) {
			id = id[:len(id)-2]
		}
		id += suffixEtched
	} else if foil && hasNonfoil {
		if strings.HasSuffix(id, suffixFoil) || strings.HasSuffix(id, suffixEtched) {
			id = id[:len(id)-2]
		}
		id += suffixFoil
	}
	return id
}

// hasOversizedPrinting reports whether the datastore carries an oversized
// printing of the card. A listing that says oversize is asking for one, and
// the sheets that were never built - the championship prizes, the oversized
// dungeons - hold nothing it could mean, so the word marks it unsupported.
//
// The sets that carry them used to be named by the words in their titles,
// which read "Commander Legends: Battle for Baldur's Gate" as an oversized
// Commander product and priced its dungeon as the ordinary token. The edition
// is the wrong thing to ask: a storefront writes "Magic Player Rewards" for a
// card filed under "Magic Player Rewards 2009", and a set can hold one
// oversized printing among ordinary ones. Ask whether the card has such a
// printing at all, and leave which one to the edition filter below.
func (b *Backend) hasOversizedPrinting(name string) bool {
	printings, err := b.Printings4Card(name)
	if err != nil {
		return false
	}
	for _, code := range printings {
		set, found := b.Sets[code]
		if !found {
			continue
		}
		for i := range set.Cards {
			if set.Cards[i].IsOversized && Equals(set.Cards[i].Name, name) {
				return true
			}
		}
	}
	return false
}

// editionFilesTokens reports whether an edition names a set of tokens. It is
// asked before a name is resolved, to tell which of two names sharing a
// normalized form the edition means.
//
// The set is looked up directly rather than through GetSetByName, whose last
// resort is to run the whole edition fixup: a sheet of tokens answers to its
// own name or its own code, and paying for the fixup to learn otherwise would
// charge every card that happens to share a name with a token.
func (b *Backend) editionFilesTokens(edition string) bool {
	if edition == "" {
		return false
	}
	set, found := b.NormalizedSets[Normalize(edition)]
	if !found {
		var err error
		set, err = b.GetSet(edition)
		if err != nil {
			return false
		}
	}
	return set.Type == "token"
}
