package pokemon

import (
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for Pokemon. A card is identified by
// name + collector number, with the qualifier label breaking ties: the same
// number recurs as several products differing only by the treatment or the
// promotion TCGplayer decorates the name with, and that label is the only
// thing between them.
//
// The foil treatment is real foilness here, unlike the Yu-Gi-Oh print runs:
// a storefront's foil flag has to reach the Holofoil printing. So the flag
// resolves through the loader's FoilUUIDs, and only an input naming a
// treatment outright re-keys onto the exact crossing it names.
type Rules struct{}

// fullNumberRe matches the game's collector number shapes: "001/102",
// "SWSH001", "TG01/TG30", "H1/H32", with the set total that follows a slash
// kept out of the capture.
var fullNumberRe = regexp.MustCompile(`(?i)\b([A-Z]{0,4}\d+[a-z]?)(?:/\d+)?\b`)

// Prefilter splits the parenthetical decorations off the name before the
// canonical-name lookup: TCGplayer writes "Pikachu (Cosmos Holo)" and
// "Charizard (Prerelease)". A full name that is itself canonical stays as it
// is — a handful of real card names carry a parenthetical of their own.
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	if !strings.Contains(inCard.Name, "(") {
		return
	}
	vars := mtgmatcher.SplitVariants(inCard.Name)
	if len(vars) > 1 {
		inCard.Name = vars[0]
		inCard.AddToVariant(strings.Join(vars[1:], " "))
	}
}

// AdjustName provides a prefix fallback for truncated feeds, adopting the
// one name among the prefix matches that carries the input's number. Names
// compare normalized, so punctuation variants of one name are not read as
// ambiguity.
func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	number := extractNumber(inCard.Variation)
	if number == "" {
		return
	}
	uuids, err := b.SearchHasPrefix(inCard.Name)
	if err != nil {
		return
	}
	name := ""
	for _, uuid := range uuids {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		if !numberMatches(number, co.Number) {
			continue
		}
		if name != "" && mtgmatcher.Normalize(name) != mtgmatcher.Normalize(co.Name) {
			return
		}
		name = co.Name
	}
	if name != "" {
		inCard.Name = name
	}
}

// AdjustEdition trims the game-name prefix and "Singles" suffix storefronts
// decorate set names with, and drops the headings that name no set of ours.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := strings.TrimSpace(inCard.Edition)
	for _, prefix := range []string{"Pokemon TCG", "Pokemon", "Pokémon"} {
		if strings.HasPrefix(edition, prefix) {
			trimmed := strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			// A set really named for the game keeps its name: "Pokemon Go"
			// and "Pokemon Futsal Promos" are sets, not decorations.
			if _, err := b.GetSetByName(trimmed); err == nil {
				edition = trimmed
			}
			break
		}
	}
	edition = strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))
	// The generic promo headings resolve to no set on purpose: the heading
	// spans every promotional set the game has - and it has more than
	// thirty - so it can gate the promo names but must not narrow the
	// candidates to whichever one happens to wear the name.
	if promoHeadings[mtgmatcher.Normalize(edition)] {
		edition = ""
	}
	inCard.Edition = edition
}

// promoHeadings are the headings storefronts file promotional printings
// under without saying which set issued them.
var promoHeadings = map[string]bool{
	mtgmatcher.Normalize("Promo"):             true,
	mtgmatcher.Normalize("Promos"):            true,
	mtgmatcher.Normalize("Promo Cards"):       true,
	mtgmatcher.Normalize("Promotional"):       true,
	mtgmatcher.Normalize("Promotionals"):      true,
	mtgmatcher.Normalize("Promotional Cards"): true,
}

func (Rules) FilterPrintings(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, editions []string) []string {
	return editions
}

// FilterCards narrows candidates by edition, collector number and label, in
// that order. A bare input facing several labels keeps them all and surfaces
// as an aliasing error rather than a guess.
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	number := extractNumber(inCard.Variation)

	var candidates []mtgmatcher.Card
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		card := co.Card

		// A product's printing siblings all file under the name bucket;
		// fold them onto their shared product id so each candidate appears
		// exactly once, and output() picks the printing afterwards. The
		// loader writes each entry's uuid onto its Card, which rules the
		// uuid out as the folding key.
		key := card.Identifiers["tcgplayerProductId"]
		if key == "" {
			key = trimFinishSuffix(uuid)
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		if _, found := cardSet[card.SetCode]; !found {
			continue
		}
		if number != "" && !numberMatches(number, card.Number) {
			continue
		}
		// An input naming a treatment re-keys the copy's FoilUUIDs so the
		// flag-driven resolution downstream lands on that printing. Both
		// slots move together: the input named the printing outright, and
		// a storefront's foil flag is the less reliable of the two.
		uuid := finishUUID(b, inCard, &card)
		if uuid != "" {
			foilUUIDs := make(map[string]string, len(card.FoilUUIDs))
			for k, v := range card.FoilUUIDs {
				foilUUIDs[k] = v
			}
			foilUUIDs[mtgmatcher.FinishNonfoil] = uuid
			foilUUIDs[mtgmatcher.FinishFoil] = uuid
			card.FoilUUIDs = foilUUIDs
		}
		candidates = append(candidates, card)
	}
	if len(candidates) <= 1 {
		return candidates
	}

	// A verbatim collector number beats a prefix-folded match.
	if number != "" {
		var exact []mtgmatcher.Card
		for _, card := range candidates {
			if strings.EqualFold(number, card.Number) {
				exact = append(exact, card)
			}
		}
		if len(exact) > 0 && len(exact) < len(candidates) {
			candidates = exact
		}
		if len(candidates) <= 1 {
			return candidates
		}
	}

	return tierByLabel(inCard, candidates)
}

// tierByLabel splits the candidates into the ones whose label the input's
// variation describes, the base printings, and the labelled printings. Only
// the variation is consulted: set names carry the label words themselves
// ("Team Plasma" is a label and part of three set names).
func tierByLabel(inCard *mtgmatcher.InputCard, candidates []mtgmatcher.Card) []mtgmatcher.Card {
	var described, base, labelled []mtgmatcher.Card
	for _, card := range candidates {
		if len(card.PromoTypes) == 0 {
			base = append(base, card)
			continue
		}
		labelled = append(labelled, card)
		// The tag is a token, so the wording's words are joined back up a
		// run at a time to ask whether they name it.
		for _, promoType := range card.PromoTypes {
			if mtgmatcher.SlugDescribes(inCard.Variation, promoType) {
				described = append(described, card)
				break
			}
		}
	}
	if len(described) > 0 {
		return described
	}
	if len(base) > 0 {
		return base
	}
	return labelled
}

// finishUUID resolves the entry an input's finish names, from the finish
// field where a caller fills one in and from the variation wording where
// only the listing speaks.
func finishUUID(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, card *mtgmatcher.Card) string {
	if uuid := b.FinishUUID(card, inCard.Finish); uuid != "" {
		return uuid
	}
	return card.FoilUUIDs[selectFinish(inCard, card)]
}

// selectFinish maps the input wording's treatment tokens onto one of the
// card's stored printings, so a listing spelling its printing out ("Reverse
// Holo", "1st Edition") resolves to that entry instead of the default one.
//
// The wording names axes rather than a printing: it can say a run, a
// treatment, or both, and the printing that answers is the stored one naming
// everything the wording asked for and the least beside it. That is what
// lets "1st Edition" reach the 1st Edition Holofoil on a card sold in no
// other first-edition printing, while still preferring the plain 1st Edition
// on a card that has one.
//
// Only the variation speaks: set names carry the same words ("Unlimited" and
// "1st Edition" name the base-set reprints). A wording naming no axis, or a
// printing the product was not priced in, keeps the flag-driven default.
func selectFinish(inCard *mtgmatcher.InputCard, card *mtgmatcher.Card) string {
	words := strings.Fields(strings.ToLower(inCard.Variation))

	var wanted []string
	if hasAllTokens(words, []string{"reverse"}) {
		wanted = append(wanted, "reverse")
	}
	if hasAllTokens(words, []string{"1st"}) || hasAllTokens(words, []string{"first"}) {
		wanted = append(wanted, finish1stEdition)
	}
	if hasAllTokens(words, []string{"unlimited"}) {
		wanted = append(wanted, finishUnlimited)
	}
	if hasAllTokens(words, []string{"holo"}) {
		wanted = append(wanted, finishHolofoil)
	}
	if len(wanted) == 0 {
		return ""
	}

	best := ""
	for key := range card.FoilUUIDs {
		// The shared slots are the defaults this is trying to move off.
		if key == mtgmatcher.FinishNonfoil || key == mtgmatcher.FinishFoil {
			continue
		}
		named := true
		for _, axis := range wanted {
			if !strings.Contains(key, axis) {
				named = false
				break
			}
		}
		if !named {
			continue
		}
		if best == "" || len(key) < len(best) {
			best = key
		}
	}
	return best
}

// hasAllTokens reports whether every token is carried by some word of the
// wording. A token matches a word it opens, so "holo" is found in "Holofoil"
// and "1st" in "1st".
func hasAllTokens(words, tokens []string) bool {
	for _, token := range tokens {
		found := false
		for _, word := range words {
			if strings.HasPrefix(word, token) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// IsUnsupported drops the products TCGplayer files under its "Cards" product
// type that are not cards a query can mean.
func (Rules) IsUnsupported(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	return false
}

func (Rules) IsSpecificUnsupported(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	return false
}

// MissingPromoTag reports an input claiming a label the resolved printing
// does not carry, so a claimed-but-absent treatment reads as unsupported
// rather than as a mismatch.
func (Rules) MissingPromoTag(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, co *mtgmatcher.CardObject) bool {
	return false
}

// CanonicalFinish places the crossings of the two axes the catalog prices.
// The plain printings belong to the shared vocabulary — "Normal" is nonfoil
// everywhere TCGplayer writes it — and the treatments past it are this
// game's own, spelled with both axes so neither the run nor the treatment is
// lost.
func (Rules) CanonicalFinish(name string) string {
	return canonicalFinish(name)
}

func canonicalFinish(name string) string {
	normalized := mtgmatcher.NormalizeFinish(name)
	switch normalized {
	case finishHolofoil, finishReverseHolofoil,
		finish1stEdition, finishUnlimited,
		finish1stEditionHolo, finishUnlimitedHolo:
		return normalized
	}
	// "Reverse Holo" and "1st Edition Holo" are how storefronts abbreviate
	// two of them, and the abbreviation names no other printing.
	switch normalized {
	case "reverseholo":
		return finishReverseHolofoil
	case "holo":
		return finishHolofoil
	case "1steditionholo":
		return finish1stEditionHolo
	case "unlimitedholo":
		return finishUnlimitedHolo
	}
	return mtgmatcher.CanonicalFinish(normalized)
}

// extractNumber pulls the collector number out of a storefront's variation
// wording, which mixes it with treatment words and set totals.
func extractNumber(variation string) string {
	for _, field := range strings.Fields(variation) {
		if m := fullNumberRe.FindStringSubmatch(field); m != nil {
			return m[1]
		}
	}
	return ""
}

// numberMatches compares a storefront's collector number against the
// catalog's, which carries the set total the storefront usually drops
// ("001/102" against "001", and either against "1").
func numberMatches(input, number string) bool {
	if number == "" {
		return false
	}
	return foldNumber(input) == foldNumber(number)
}

// foldNumber reduces a collector number to the letters and digits that carry
// it, dropping the set total and the zeros each digit run is padded with.
func foldNumber(number string) string {
	number = strings.Split(number, "/")[0]
	var out strings.Builder
	digits := false
	for _, r := range strings.ToLower(number) {
		switch {
		case r >= '0' && r <= '9':
			if !digits && r == '0' {
				continue
			}
			digits = true
			out.WriteRune(r)
		case r >= 'a' && r <= 'z':
			digits = false
			out.WriteRune(r)
		}
	}
	return out.String()
}
