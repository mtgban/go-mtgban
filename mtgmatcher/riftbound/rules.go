package riftbound

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for Riftbound. Like Lorcana, a card
// is identified by name + collector number + foil, with the edition breaking
// ties when it resolves; there are no variant tables or promo types, so most
// hooks are no-ops and the real work is the number disambiguation in
// FilterCards (foil is honored downstream by output).
type Rules struct{}

// Prefilter splits a trailing parenthetical variant off the name before the
// canonical-name lookup — unless the full name is itself a known card: a few
// real names carry a parenthetical ("Recruit (DE)"), and the promotional
// printings keep their storefront names verbatim, qualifiers included.
// Dashes stay intact, since they occur in real names too ("Dark Child -
// Starter").
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	// Names that only promotional printings carry act as unknown unless the
	// input targets a promo set: promos keep their storefront names verbatim
	// (qualifiers included), and those shapes must keep resolving to the
	// main printings for everyone else.
	targetsPromo := editionIsPromo(b, inCard.Edition)
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		if !promoOnlyName(b, inCard.Name) || targetsPromo {
			return
		}
	}
	if strings.Contains(inCard.Name, "(") {
		vars := mtgmatcher.SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.AddToVariant(strings.Join(vars[1:], " "))
		}
	}
	// A promo printing can still share the storefront shape of a main-set
	// legend name ("Teemo - Swift Scout" is a promo entry while the Origins
	// card is "Swift Scout"): re-aim at the gallery name here, or the
	// canonical lookup would stop at the promo, whose printings the promo
	// gate in FilterCards then rightly refuses.
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found &&
		promoOnlyName(b, inCard.Name) && !targetsPromo {
		if fixed := legendName(b, inCard.Name); fixed != "" {
			inCard.Name = fixed
		}
	}
}

// editionIsPromo reports whether the input edition resolves to one of the
// promotional sets.
func editionIsPromo(b *mtgmatcher.Backend, edition string) bool {
	if edition == "" {
		return false
	}
	set, err := b.GetSetByName(edition)
	if err != nil {
		return false
	}
	return set.Type == "promo"
}

// promoOnlyName reports whether every printing hashed under the name lives
// in a promotional set.
func promoOnlyName(b *mtgmatcher.Backend, name string) bool {
	uuids := b.Hashes[mtgmatcher.Normalize(name)]
	if len(uuids) == 0 {
		return false
	}
	for _, uuid := range uuids {
		co, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		set, found := b.Sets[co.SetCode]
		if !found || set.Type != "promo" {
			return false
		}
	}
	return true
}

// AdjustName reconciles storefront name shapes with the gallery's. Legend
// cards are exported title-only ("Daughter of the Void") while storefronts
// list them champion-first ("Kai'Sa - Daughter of the Void"), so an unknown
// name containing " - " retries as its title segment; the collector number
// and finish still validate the outcome downstream. Failing that, a prefix
// fallback covers feeds that truncate a "Champion, Title" name: scan for
// cards whose name has the input as a prefix and let the number and finish
// narrow them, adopting the name only when exactly one survives.
func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}

	if fixed := legendName(b, inCard.Name); fixed != "" {
		inCard.Name = fixed
		return
	}

	uuids, err := b.SearchHasPrefix(inCard.Name)
	if err != nil {
		return
	}

	number := extractNumber(inCard.Variation)
	var match string
	for _, uuid := range uuids {
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		if number != "" && !strings.EqualFold(number, co.Number) {
			continue
		}
		if inCard.Foil && !co.HasFinish(mtgmatcher.FinishFoil) {
			continue
		}
		if !inCard.Foil && !co.HasFinish(mtgmatcher.FinishNonfoil) {
			continue
		}
		if match != "" && match != co.Name {
			// Different names survive the filters: genuinely ambiguous.
			return
		}
		match = co.Name
	}
	if match != "" {
		inCard.Name = match
	}
}

// AdjustEdition normalizes scraper edition strings toward the gallery set
// names: storefronts commonly prefix the game name ("Riftbound: Origins")
// or append catalog suffixes ("... Singles"). An edition that still matches
// no set name simply does not narrow the candidates (the Match skeleton
// falls back to every printing), so trimming can only help.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := strings.TrimSpace(inCard.Edition)
	// An edition already naming a set verbatim needs no normalization — and
	// must not be trimmed out of matching: the promotional sets themselves
	// are named "Riftbound ... Promotional Cards".
	for _, set := range b.Sets {
		if mtgmatcher.Equals(set.Name, edition) {
			inCard.Edition = edition
			return
		}
	}
	for _, prefix := range []string{"Riftbound: League of Legends", "Riftbound", "League of Legends"} {
		if strings.HasPrefix(edition, prefix) {
			edition = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			break
		}
	}
	edition = strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))
	if name, found := storefrontEditions[mtgmatcher.Normalize(edition)]; found {
		edition = name
	}
	inCard.Edition = edition
}

// storefrontEditions renames the editions storefronts carry that the gallery
// names differently. Without them the edition narrows nothing and the number
// alone decides, which quietly answers with the base-set printing: a listing
// of the Top 8 Guardian Angel, numbered 51 in the promotional set, comes back
// as the Spiritforged card of the same number.
//
// Only the renames the data agrees on are here. Every product id that
// resolves in these three lands in one set apiece - 31, 58 and 24 of them -
// whereas CardTrader's plain "Promos" splits across two promotional sets, so
// there is no single answer to give it and it is deliberately absent.
var storefrontEditions = map[string]string{
	mtgmatcher.Normalize("Organized Play"):          "Riftbound Organized Play Promotional Cards",
	mtgmatcher.Normalize("Nexus Night Promos"):      "Riftbound Organized Play Promotional Cards",
	mtgmatcher.Normalize("Origins Proving Grounds"): "Proving Grounds",
}

// legendName maps a storefront "Champion - Title" name onto the gallery name
// it stands for, or returns "" when no unambiguous mapping exists. The
// gallery exports Legend cards title-only ("Daughter of the Void"), may
// shorten the champion ("Master Yi - Meditative" is "Yi, Meditative"), and
// names starter legends title-first ("Lux - Lady of Luminosity (Starter)" is
// "Lady of Luminosity - Starter"): accept the bare title, a "Champion,
// Title" name whose title matches and whose champion ends the input's
// champion, or a dashed name led by the title — as long as exactly one
// candidate does overall.
func legendName(b *mtgmatcher.Backend, name string) string {
	champion, title, found := strings.Cut(name, " - ")
	if !found {
		return ""
	}
	if _, known := b.CanonicalNames[mtgmatcher.Normalize(title)]; known {
		return title
	}
	var match string
	for _, canonical := range b.AllCanonicalNames {
		c, t, ok := strings.Cut(canonical, ", ")
		matches := ok && mtgmatcher.Equals(t, title) &&
			strings.HasSuffix(mtgmatcher.Normalize(champion), mtgmatcher.Normalize(c))
		if !matches {
			t2, _, ok2 := strings.Cut(canonical, " - ")
			matches = ok2 && mtgmatcher.Equals(t2, title)
		}
		if !matches {
			continue
		}
		if match != "" && match != canonical {
			return ""
		}
		match = canonical
	}
	return match
}

func (Rules) FilterPrintings(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, editions []string) []string {
	return editions
}

func (Rules) IsUnsupported(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	return false
}

func (Rules) IsSpecificUnsupported(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	return false
}

func (Rules) MissingPromoTag(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, co *mtgmatcher.CardObject) bool {
	return false
}

// FilterCards narrows candidates by edition, collector number, and finish,
// mirroring the Lorcana rules: candidates come from the name hash (stable
// load order), the cardSet keys carry the sets matching the input edition
// when one was supplied and resolves (falling back to every printing
// otherwise), and the number comparison is case-insensitive because real
// numbers carry letter affixes ("66a", "T5", "SP3").
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	number := extractNumber(inCard.Variation)

	// Promotional printings reuse the main sets' collector numbers, so they
	// never match implicitly: only an edition that resolves to a promo set
	// reaches them, and everything else keeps matching the main printings.
	allowPromo := editionIsPromo(b, inCard.Edition)

	var out []mtgmatcher.Card
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		// Foil printings are stored under a "_f"-suffixed uuid; fold them
		// back onto the base card so each candidate appears exactly once.
		// Base uuids never contain underscores, so the first underscore
		// marks the start of the finish suffix.
		base := uuid
		if idx := strings.IndexByte(uuid, '_'); idx >= 0 {
			base = uuid[:idx]
		}
		if seen[base] {
			continue
		}
		seen[base] = true

		co, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		card := co.Card

		if _, found := cardSet[card.SetCode]; !found {
			continue
		}
		if !allowPromo {
			if set, found := b.Sets[card.SetCode]; found && set.Type == "promo" {
				continue
			}
		}
		if number != "" && !strings.EqualFold(number, card.Number) {
			continue
		}
		// Every card carries both finishes today, so this never drops a
		// candidate; it stays for symmetry with Lorcana should finish data
		// ever appear in the gallery.
		if inCard.Foil && !card.HasFinish(mtgmatcher.FinishFoil) {
			continue
		}
		if !inCard.Foil && !card.HasFinish(mtgmatcher.FinishNonfoil) {
			continue
		}
		out = append(out, card)
	}
	return out
}

// extractNumber pulls the collector number out of the scraper-supplied
// Variation. Core Match may append parenthetical chunks split off the input
// name, so only the first field containing a digit counts — numbers may be
// letter-prefixed ("T1", "SP3"), letter-suffixed ("66a"), or starred
// ("227*"). A full public code ("OGN-066a/298") reduces to its number, and
// the result is canonicalized exactly like the loader's numbers so
// zero-padded feeds compare equal.
//
// A trailing "s" becomes the star: storefronts number the signed showcase
// printings "302s" where the gallery numbers them "302*". The two never
// collide, since no published number ends in a letter other than a, b or c.
func extractNumber(variation string) string {
	number := ""
	for _, field := range strings.Fields(variation) {
		if strings.ContainsAny(field, "0123456789") {
			number = field
			break
		}
	}
	if idx := strings.LastIndexByte(number, '-'); idx >= 0 {
		number = number[idx+1:]
	}
	number = strings.Split(number, "/")[0]
	number = CanonicalNumber(number)
	if len(number) > 1 && (number[len(number)-1] == 's' || number[len(number)-1] == 'S') {
		if last := number[len(number)-2]; last >= '0' && last <= '9' {
			number = number[:len(number)-1] + "*"
		}
	}
	return number
}
